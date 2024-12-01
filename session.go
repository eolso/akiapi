package akiapi

type SessionManager interface {
	NewGame(theme Theme, childMode bool) error

	Respond(response Response) error
	UndoResponse() error

	Question() string
	Answer() Answer
	Progress() string
	History() []QuestionAnswer

	IsAnswered() bool
	AcceptAnswer() error
	DeclineAnswer() error
}

type Theme string

const (
	ThemeCharacters Theme = "1"
	ThemeObjects    Theme = "2"
	ThemeAnimals    Theme = "14"
)

type Response string

const (
	ResponseYes         Response = "0"
	ResponseNo          Response = "1"
	ResponseDontKnow    Response = "2"
	ResponseProbably    Response = "3"
	ResponseProbablyNot Response = "4"
)

type Answer struct {
	Id          string
	Name        string
	Description string
	PhotoUrl    string
}

type QuestionAnswer struct {
	Question string
	Answer   Response
}
