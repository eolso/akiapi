package akiapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Language string

const (
	English Language = "en"
	Spanish          = "es"
)

type Game interface {
	// Question returns the current question of the game
	Question() string
	// Options returns the response options available for the current question
	Options() []string
	// SelectOption submits int index of Options as the response to the Question
	SelectOption(int) error
	// Undo rescinds the most recent response sent by SelectOption
	Undo() error
	// Guess returns the current highest probability guess
	Guess() (Guess, error)
	// ListGuesses returns all the current guesses with > 0 probability
	ListGuesses() ([]Guess, error)
	// Progress returns the current highest probability amongst the guesses
	Progress() float64
	// Responses returns the slice of Response selected this game
	Responses() []Response
}

type GameOptions struct {
	Language  Language
	Theme     Theme
	ChildMode bool
}

type Response struct {
	Question string
	Answer   string
}

type game struct {
	options   GameOptions
	responses []Response
	lastStep  stepInfo
	urlValues url.Values
}

func NewGame(opt GameOptions) (Game, error) {
	g := &game{options: opt, urlValues: url.Values{}}

	// Check if theme is set. If it isn't, default to CharactersTheme.
	if opt.Theme == nil {
		g.options.Theme = CharactersTheme
	}

	// Verify that CharactersTheme was initialized successfully
	if opt.Theme == nil {
		return nil, fmt.Errorf("no theme supplied")
	}

	err := g.connect()
	if err != nil {
		return nil, err
	}

	err = g.newSession()

	return g, err
}

func (g *game) Question() string {
	return g.lastStep.Question
}

func (g *game) Options() []string {
	answers := make([]string, 0, 5)

	for _, answer := range g.lastStep.Answers {
		answers = append(answers, answer.Answer)
	}

	return answers
}

func (g *game) SelectOption(i int) error {
	return g.selectOption(i)
}

func (g *game) Undo() error {
	return g.undo()
}

func (g *game) Guess() (Guess, error) {
	guesses, err := g.listGuesses()
	if err != nil {
		return nil, err
	}

	if len(guesses) == 0 {
		return nil, fmt.Errorf("no guesses available")
	}

	return guesses[0], nil
}

func (g *game) ListGuesses() ([]Guess, error) {
	return g.listGuesses()
}

func (g *game) Progress() float64 {
	progress, _ := strconv.ParseFloat(g.lastStep.Progression, 64)
	return progress
}

func (g *game) Responses() []Response {
	return g.responses
}

func (g *game) connect() error {
	resp, err := httpClient.Get(fmt.Sprintf(gameUrlFmt, g.options.Language))
	if err != nil {
		return err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	uidRegex, err := regexp.Compile("var uid_ext_session = '(.*)'")
	if err != nil {
		return err
	}

	addrRegex, err := regexp.Compile("var frontaddr = '(.*)'")
	if err != nil {
		return err
	}

	if match := uidRegex.FindStringSubmatch(string(b)); len(match) > 1 {
		g.urlValues.Set("uid_ext_session", match[1])
	} else {
		return fmt.Errorf("could not locate uid_ext_session")
	}

	if match := addrRegex.FindStringSubmatch(string(b)); len(match) > 1 {
		g.urlValues.Set("frontaddr", match[1])
	} else {
		return fmt.Errorf("could not locate frontaddr")
	}

	g.urlValues.Set("answer", "")
	g.urlValues.Set("callback", "")
	g.urlValues.Set("childMod", strconv.FormatBool(g.options.ChildMode))
	g.urlValues.Set("constraint", "ETAT<>'AV'")
	g.urlValues.Set("partner", "1")
	g.urlValues.Set("player", "website-desktop")
	g.urlValues.Set("question_filter", "''")
	g.urlValues.Set("soft_constaint", "''")
	g.urlValues.Set("urlApiWs", g.options.Theme.Url())

	return nil
}

func (g *game) newSession() error {
	g.urlValues.Set("callback", fmt.Sprintf("jQuery%d", time.Now().UnixNano()))

	u := fmt.Sprintf(baseUrlFmt, g.options.Language) + "/new_session?" + g.urlValues.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// The response is a json response wrapped by jQuery<id>(), so we'll need to strip that off before decoding.
	responseBytes := b[strings.Index(string(b), "(")+1 : len(b)-1]

	var response sessionResponse
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return err
	}

	g.urlValues.Set("signature", response.Parameters.Identification.Signature)
	g.urlValues.Set("step", response.Parameters.StepInformation.Step)
	g.urlValues.Set("session", response.Parameters.Identification.Session)

	g.lastStep = response.Parameters.StepInformation

	return nil
}

func (g *game) selectOption(i int) error {
	if i < 0 || i >= len(g.lastStep.Answers) {
		return fmt.Errorf("answer selection is invalid")
	}

	r := Response{
		Question: g.Question(),
		Answer:   g.Options()[i],
	}

	g.urlValues.Set("callback", fmt.Sprintf("jQuery%d", time.Now().UnixNano()))
	g.urlValues.Set("answer", strconv.Itoa(i))

	u := fmt.Sprintf(baseUrlFmt, g.options.Language) + "/answer_api?" + g.urlValues.Encode()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit answer: %w", err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return fmt.Errorf("failed to submit answer: %w", err)
	}

	// The response is a json response wrapped by jQuery<id>(), so we'll need to strip that off before decoding.
	responseBytes := b[strings.Index(string(b), "(")+1 : len(b)-1]

	var response answerResponse
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return fmt.Errorf("failed to submit answer: %w", err)
	}

	g.lastStep = response.Parameters
	g.urlValues.Set("step", g.lastStep.Step)
	g.responses = append(g.responses, r)

	return nil
}

func (g *game) undo() error {
	if g.lastStep.Step == "0" {
		return nil
	}

	g.urlValues.Set("callback", fmt.Sprintf("jQuery%d", time.Now().UnixNano()))
	g.urlValues.Set("answer", "-1")

	u := fmt.Sprintf(g.options.Theme.Url()) + "/cancel_answer?" + g.urlValues.Encode()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to undo answer: %w", err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return fmt.Errorf("failed to undo answer: %w", err)
	}

	// The response is a json response wrapped by jQuery<id>(), so we'll need to strip that off before decoding.
	responseBytes := b[strings.Index(string(b), "(")+1 : len(b)-1]

	var response answerResponse
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return fmt.Errorf("failed to undo answer: %w", err)
	}

	g.lastStep = response.Parameters
	g.urlValues.Set("step", g.lastStep.Step)
	g.responses = g.responses[:len(g.responses)-1]

	return nil
}

func (g *game) listGuesses() ([]Guess, error) {
	g.urlValues.Set("callback", fmt.Sprintf("jQuery%d", time.Now().UnixNano()))

	u := fmt.Sprintf(g.options.Theme.Url()) + "/list?" + g.urlValues.Encode()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list guesses: %w", err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return nil, fmt.Errorf("failed to list guesses: %w", err)
	}

	// The response is a json response wrapped by jQuery<id>(), so we'll need to strip that off before decoding.
	responseBytes := b[strings.Index(string(b), "(")+1 : len(b)-1]

	var response listResponse
	if err = json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to list guesses: %w", err)
	}

	guesses := make([]Guess, len(response.Parameters.Elements))

	for i, e := range response.Parameters.Elements {
		guesses[i] = e.Element
	}

	return guesses, nil
}
