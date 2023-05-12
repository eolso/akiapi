package akiapi

import (
	"net/http"
	"strconv"
)

const baseUrlFmt = `https://%s.akinator.com`
const gameUrlFmt = `https://%s.akinator.com/game`

var httpClient = &http.Client{}

type Guess interface {
	Name() string
	Description() string
	Probability() float64
	Image() string
}

type sessionResponse struct {
	Completion string `json:"completion"`
	Parameters struct {
		Identification struct {
			Channel       int    `json:"channel"`
			Session       string `json:"session"`
			Signature     string `json:"signature"`
			ChallengeAuth string `json:"challenge_auth"`
		} `json:"identification"`
		StepInformation stepInfo `json:"step_information"`
	} `json:"parameters"`
}

type answerResponse struct {
	Completion string   `json:"completion"`
	Parameters stepInfo `json:"parameters"`
}

type element struct {
	Id                  string `json:"id"`
	EName               string `json:"name"`
	IdBase              string `json:"id_base"`
	EProbability        string `json:"proba"`
	EDescription        string `json:"description"`
	ValidContraint      string `json:"valide_contrainte"`
	Ranking             string `json:"ranking"`
	Pseudo              string `json:"pseudo"`
	PicturePath         string `json:"picture_path"`
	Corrupt             string `json:"corrupt"`
	Relative            string `json:"relative"`
	AwardID             string `json:"award_id"`
	FlagPhoto           int    `json:"flag_photo"`
	AbsolutePicturePath string `json:"absolute_picture_path"`
}

type listResponse struct {
	Completion string `json:"completion"`
	Parameters struct {
		Elements []struct {
			Element element `json:"element"`
		} `json:"elements"`
		NumElements string `json:"NbObjetsPertinents"`
	} `json:"parameters"`
}

type stepInfo struct {
	Question string `json:"question"`
	Answers  []struct {
		Answer string `json:"answer"`
	} `json:"answers"`
	Step           string `json:"step"`
	Progression    string `json:"progression"`
	QuestionId     string `json:"questionid"`
	InfoGain       string `json:"infogain"`
	StatusMinibase string `json:"status_minibase"`
	Options        any    `json:"options"`
}

func SetHttpClient(c *http.Client) {
	httpClient = c
	initThemes()

}

func (e element) Name() string {
	return e.EName
}

func (e element) Description() string {
	return e.Description()
}

func (e element) Probability() float64 {
	p, _ := strconv.ParseFloat(e.EProbability, 64)
	return p * 100
}

func (e element) Image() string {
	return e.AbsolutePicturePath
}
