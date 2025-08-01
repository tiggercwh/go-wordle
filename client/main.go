package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode"
)

const maxRounds = 6
const serverURL = "http://localhost:8080/api"

type Score int

const (
	Miss Score = iota
	Present
	Hit
	Unknown = -1
)

type LetterResult struct {
	Char  rune  `json:"char"`
	Score Score `json:"score"`
}

type GameState struct {
	ID           string           `json:"id"`
	Round        int              `json:"round"`
	MaxRounds    int              `json:"maxRounds"`
	History      [][]LetterResult `json:"history"`
	Candidates   []string         `json:"candidates"`
	GameOver     bool             `json:"gameOver"`
	Won          bool             `json:"won"`
	CreatedAt    string           `json:"createdAt"`
	LastActivity string           `json:"lastActivity"`
}

type GuessRequest struct {
	Word string `json:"word"`
}

type GuessResponse struct {
	Success   bool           `json:"success"`
	Message   string         `json:"message"`
	Result    []LetterResult `json:"result,omitempty"`
	GameState *GameState     `json:"gameState,omitempty"`
	GameOver  bool           `json:"gameOver"`
	Won       bool           `json:"won"`
}

type NewGameResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	GameState GameState `json:"gameState"`
}

func makeRequest(method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func createNewGame() (*GameState, error) {
	respBody, err := makeRequest("POST", serverURL+"/game/new", nil)
	if err != nil {
		return nil, err
	}

	var response NewGameResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, fmt.Errorf("failed to create game: %s", response.Message)
	}

	return &response.GameState, nil
}

func submitGuess(gameID, word string) (*GuessResponse, error) {
	request := GuessRequest{Word: word}
	respBody, err := makeRequest("POST", fmt.Sprintf("%s/game/%s/guess", serverURL, gameID), request)
	if err != nil {
		return nil, err
	}

	var response GuessResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func printGuessResult(result []LetterResult) {
	for _, r := range result {
		switch r.Score {
		case Hit:
			fmt.Printf("\033[1;32m%c\033[0m", unicode.ToUpper(r.Char)) // green
		case Present:
			fmt.Printf("\033[1;33m%c\033[0m", unicode.ToUpper(r.Char)) // yellow
		case Miss:
			fmt.Printf("\033[1;90m%c\033[0m", unicode.ToUpper(r.Char)) // dim grey
		}
	}
	fmt.Println()
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to Wordle CLI Client!")

	// For Debugging
	// fmt.Printf("Connecting to server at %s\n", serverURL)

	gameState, err := createNewGame()
	if err != nil {
		fmt.Printf("Error creating game: %v\n", err)
		fmt.Println("Make sure the server is running at http://localhost:8080")
		return
	}

	// For Debugging
	// fmt.Printf("Game created! Game ID: %s\n", gameState.ID)
	// fmt.Printf("Round %d/%d\n", gameState.Round, gameState.MaxRounds)

	// Game loop
	for round := 1; round <= maxRounds; round++ {
		fmt.Printf("\nRound %d/%d\n", round, maxRounds)

		// Display history
		for _, past := range gameState.History {
			printGuessResult(past)
		}

		fmt.Print("Enter a 5-letter word: ")
		scanner.Scan()
		guess := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if len(guess) != 5 {
			fmt.Println("Please enter a valid 5-letter word.")
			round--
			continue
		}

		// Submit guess to server
		response, err := submitGuess(gameState.ID, guess)
		if err != nil {
			fmt.Printf("Error submitting guess: %v\n", err)
			round--
			continue
		}

		if !response.Success {
			fmt.Printf("Guess failed: %s\n", response.Message)
			round--
			continue
		}

		// Update game state
		gameState = response.GameState

		// Display result
		if response.Result != nil {
			printGuessResult(response.Result)
		}

		// Check if game is over
		if response.GameOver {
			if response.Won {
				fmt.Println("Congratulations! You guessed the word.")
			} else {
				if len(gameState.Candidates) == 1 {
					fmt.Printf("Game over! The word was: %s\n", gameState.Candidates[0])
				} else {
					fmt.Println("Game over! You ran out of guesses.")
				}
			}
			return
		}
	}

	if len(gameState.Candidates) == 1 {
		fmt.Printf("Game over! The word was: %s\n", gameState.Candidates[0])
	} else {
		fmt.Println("Game over! You ran out of guesses.")
	}
}
