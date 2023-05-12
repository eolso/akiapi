package akiapi

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var (
	CharactersTheme Theme
	AnimalsTheme    Theme
	ObjectsTheme    Theme
)

type Theme interface {
	Name() string
	Url() string
}

type theme struct {
	ThemeName string `json:"translated_theme_name"`
	UrlWs     string `json:"urlWs"`
	SubjectId string `json:"subject_id"`
}

// Attempt to pre-populate the global Theme variables
func init() {
	initThemes()
}

func initThemes() {
	themes, err := GetThemes()
	if err != nil {
		return
	}

	for _, t := range themes {
		switch strings.ToLower(t.Name()) {
		case "characters":
			CharactersTheme = t
		case "animals":
			AnimalsTheme = t
		case "objects":
			ObjectsTheme = t
		}
	}
}

func GetThemes() ([]Theme, error) {
	r, err := regexp.Compile("'arrUrlThemesToPlay', (.*)\\);")
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", "https://en.akinator.com", nil)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	matched := r.FindStringSubmatch(string(b))
	if len(matched) < 1 {
		return nil, err
	}

	var respThemes []theme
	if err = json.Unmarshal([]byte(matched[1]), &respThemes); err != nil {
		return nil, err
	}

	themes := make([]Theme, len(respThemes))
	for i, t := range respThemes {
		themes[i] = t
	}

	return themes, nil
}

func (t theme) Name() string {
	return t.ThemeName
}

func (t theme) Url() string {
	return t.UrlWs
}
