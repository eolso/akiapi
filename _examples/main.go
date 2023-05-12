package main

import (
	"fmt"

	"github.com/eolso/akiapi"
)

func main() {
	// This might need to be uncommented depending on the site's status/your os' certs
	//akiapi.SetHttpClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}})

	game, err := akiapi.NewGame(akiapi.GameOptions{Theme: akiapi.CharactersTheme, Language: akiapi.English})
	if err != nil {
		panic(err)
	}

	var response int
	questionBuffer := 0
	for {
		if game.Progress() > 80.0 && questionBuffer <= 0 {
			guess, err := game.Guess()
			if err != nil {
				panic(err)
			}
			fmt.Printf("Is it: %s?\n", guess.Name())
			fmt.Println("1) Yes")
			fmt.Println("2) No")
			fmt.Scanln(&response)

			if response == 1 {
				fmt.Println("Ayy lmao")
				return
			} else if response == 2 {
				questionBuffer = 5
				continue
			}
		}

		fmt.Println(game.Question())
		for index, answer := range game.Options() {
			fmt.Printf("%d) %s\n", index+1, answer)
		}

		fmt.Printf("> ")
		fmt.Scanln(&response)

		switch response {
		case 0:
			if err = game.Undo(); err != nil {
				panic(err)
			}
		case 1, 2, 3, 4, 5:
			if err = game.SelectOption(response - 1); err != nil {
				panic(err)
			}
			questionBuffer--
		case 6:
			guesses, err := game.ListGuesses()
			if err != nil {
				panic(err)
			}

			for i, guess := range guesses {
				fmt.Printf("    %d) %s [%02f]\n", i+1, guess.Name(), guess.Probability())
			}
		case 7:
			for i, r := range game.Responses() {
				fmt.Printf("    %d) %s %s\n", i+1, r.Question, r.Answer)
			}
		}
	}
}
