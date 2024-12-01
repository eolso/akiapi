package main

import (
	"fmt"
	"strings"

	"github.com/eolso/akiapi"
)

func main() {
	var session akiapi.SessionManager
	session = akiapi.NewClient()

	if err := session.NewGame(akiapi.ThemeCharacters, false); err != nil {
		panic(err)
	}

	var response string
	for {
		if session.IsAnswered() {
			fmt.Println(session.Answer().Name)
			if err := session.AcceptAnswer(); err != nil {
				panic(err)
			}
			return
		}

		fmt.Println(session.Question())
		fmt.Printf("> ")
		fmt.Scanln(&response)

		if response == "u" {
			if err := session.UndoResponse(); err != nil {
				panic(err)
			}
		} else {
			if err := session.Respond(akiapi.Response(strings.TrimSpace(response))); err != nil {
				panic(err)
			}
		}
	}
}
