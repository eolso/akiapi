package akiapi

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var stateRegex = regexp.MustCompile(`localStorage.setItem\('([^']*)', '([^']*)'\);`)
var questionRegex = regexp.MustCompile(`<p[^>]*>([^<]*)</p>`)

const GameUrl = "https://en.akinator.com/game"

const AnswerUrl = "https://en.akinator.com/answer"
const UndoUrl = "https://en.akinator.com/cancel_answer"

const AcceptUrl = "https://en.akinator.com/choice"
const DeclineUrl = "https://en.akinator.com/exclude"

type Client struct {
	step        string
	progression string
	signature   string
	session     string
	identifier  string
	question    string

	answer  Answer
	history []QuestionAnswer

	theme     Theme
	childMode bool

	httpClient *http.Client
}

type AnswerResponse struct {
	Completion  string `json:"completion"`
	Step        string `json:"step"`
	Progression string `json:"progression"`
	QuestionID  string `json:"question_id"`
	Question    string `json:"question"`

	GuessId          string `json:"id_proposition"`
	GuessName        string `json:"name_proposition"`
	GuessDescription string `json:"description_proposition"`
	PhotoUrl         string `json:"photo"`
}

type AutoGenerated struct {
	Completion             string `json:"completion"`
	IDProposition          string `json:"id_proposition"`
	IDBaseProposition      string `json:"id_base_proposition"`
	ValideContrainte       string `json:"valide_contrainte"`
	NameProposition        string `json:"name_proposition"`
	DescriptionProposition string `json:"description_proposition"`
	FlagPhoto              string `json:"flag_photo"`
	Photo                  string `json:"photo"`
	Pseudo                 string `json:"pseudo"`
	NbElements             int    `json:"nb_elements"`
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SetHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

func (c *Client) NewGame(theme Theme, childMode bool) error {
	reqBody := strings.NewReader(fmt.Sprintf("sid=%s&cm=%t", theme, childMode))
	req, err := http.NewRequest("POST", GameUrl, reqBody)
	if err != nil {
		return fmt.Errorf("failed to build NewGame request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send NewGame request: %w", err)
	}
	defer resp.Body.Close()

	c.theme = theme
	c.childMode = childMode

	return c.readGameHtml(resp.Body)
}

func (c *Client) Respond(response Response) error {
	questionAnswer := QuestionAnswer{Question: c.question, Answer: response}

	req, err := http.NewRequest("POST", AnswerUrl, strings.NewReader(c.formatResponseBody(response)))
	if err != nil {
		return fmt.Errorf("failed to build Respond request: %w", err)
	}

	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")

	var resp *http.Response
	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Respond request: %w", err)
	}
	defer resp.Body.Close()

	var answer AnswerResponse
	if err = json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return fmt.Errorf("failed to decode Respond response: %w", err)
	}

	if answer.Completion != "OK" {
		return fmt.Errorf("invalid response received from Respond")

	}

	if len(answer.GuessName) > 0 {
		c.answer.Id = answer.GuessId
		c.answer.Name = answer.GuessName
		c.answer.Description = answer.GuessDescription
		c.answer.PhotoUrl = answer.PhotoUrl
	} else {
		c.step = answer.Step
		c.progression = answer.Progression
		c.question = answer.Question
	}

	c.history = append(c.history, questionAnswer)

	return nil
}

func (c *Client) UndoResponse() error {
	req, err := http.NewRequest("POST", UndoUrl, strings.NewReader(c.formatUndoBody()))
	if err != nil {
		return fmt.Errorf("failed to build Respond request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")

	var resp *http.Response
	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Undo request: %w", err)
	}
	defer resp.Body.Close()

	var answer AnswerResponse
	if err = json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return fmt.Errorf("failed to decode Undo response: %w", err)
	}

	c.step = answer.Step
	c.progression = answer.Progression
	c.question = answer.Question

	if len(c.history) > 0 {
		c.history = c.history[:len(c.history)-1]
	}

	return nil
}

func (c *Client) Question() string {
	return c.question
}

func (c *Client) History() []QuestionAnswer {
	return c.history
}

func (c *Client) IsAnswered() bool {
	return len(c.answer.Name) > 0
}

func (c *Client) Answer() Answer {
	return c.answer
}

func (c *Client) AcceptAnswer() error {
	req, err := http.NewRequest("POST", AcceptUrl, strings.NewReader(c.formatAcceptBody()))
	if err != nil {
		return fmt.Errorf("failed to build AcceptAnswer request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send AcceptAnswer request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response from AcceptAnswer")
	}

	return nil
}

func (c *Client) DeclineAnswer() error {
	req, err := http.NewRequest("POST", DeclineUrl, strings.NewReader(c.formatDeclineBody()))
	if err != nil {
		return fmt.Errorf("failed to build DeclineAnswer request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DeclineAnswer request: %w", err)
	}

	var answer AnswerResponse
	if err = json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return fmt.Errorf("failed to decode DeclineAnswer response: %w", err)
	}

	c.step = answer.Step
	c.progression = answer.Progression
	c.question = answer.Question
	c.answer.Id = ""
	c.answer.Name = ""
	c.answer.Description = ""
	c.answer.PhotoUrl = ""

	return nil
}

func (c *Client) readGameHtml(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "localStorage.setItem(") {
			match := stateRegex.FindStringSubmatch(line)
			if len(match) != 3 {
				continue
			}

			switch match[1] {
			case "step":
				c.step = match[2]
			case "progression":
				c.progression = match[2]
			case "signature":
				c.signature = match[2]
			case "session":
				c.session = match[2]
			case "identifiant":
				c.identifier = match[2]
			}
		} else if strings.Contains(line, `id="question-label"`) {
			match := questionRegex.FindStringSubmatch(line)
			if len(match) != 2 {
				continue
			}

			c.question = strings.ReplaceAll(match[1], `&#39;`, `'`)
		}
	}

	return c.validateState()
}

func (c *Client) formatResponseBody(response Response) string {
	return fmt.Sprintf("step=%s&progression=%s&sid=%s&cm=%t&answer=%s&step_last_proposition=&session=%s&signature=%s", c.step, c.progression, c.theme, c.childMode, response, c.session, c.signature)
}

func (c *Client) formatUndoBody() string {
	return fmt.Sprintf("step=%s&progression=%s&sid=%s&cm=%t&session=%s&signature=%s", c.step, c.progression, c.theme, c.childMode, c.session, c.signature)
}

func (c *Client) formatAcceptBody() string {
	return fmt.Sprintf("sid=%s&pid=%s&identifiant=%s&pflag_photo=1&charac_name=%s&charac_desc=%s&session=%s&signature=%s&step=%s", c.theme, c.answer.Id, c.identifier, strings.ReplaceAll(c.answer.Name, " ", "+"), strings.ReplaceAll(c.answer.Description, " ", "+"), c.session, c.signature, c.step)
}

func (c *Client) formatDeclineBody() string {
	return fmt.Sprintf("step=%s&sid=%s&cm=%t&progression=%s&session=%s&signature=%s", c.step, c.theme, c.childMode, c.progression, c.session, c.signature)
}

func (c *Client) validateState() error {
	if len(c.step) == 0 {
		return errors.New("step cannot be empty")
	} else if len(c.progression) == 0 {
		return errors.New("progression cannot be empty")
	} else if len(c.signature) == 0 {
		return errors.New("signature cannot be empty")
	} else if len(c.session) == 0 {
		return errors.New("session cannot be empty")
	} else if len(c.identifier) == 0 {
		return errors.New("identifier cannot be empty")
	} else if len(c.question) == 0 {
		return errors.New("question cannot be empty")
	}

	return nil

}
