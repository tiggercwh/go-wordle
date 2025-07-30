package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"unicode"
)

const maxRounds = 6

var wordList = []string{
	"crane", "slate", "fling", "grasp", "thing",
	"blame", "pride", "slope", "drink", "plant",
}

// LetterScore represents result for a letter
type Score int

const (
	Miss Score = iota
	Present
	Hit
	Unknown = -1
)

type LetterResult struct {
	Char  rune
	Score Score
}

func pickAnswer() string {
	// New(NewSource(seed))
	return wordList[rand.Intn(len(wordList))]
}

func scoreGuess(answer, guess string, letterStates map[rune]Score) []LetterResult {
	res := make([]LetterResult, 5)
	answerCounts := make(map[byte]int)

	// First pass: identify hits
	for i := 0; i < 5; i++ {
		if guess[i] == answer[i] {
			res[i] = LetterResult{Char: rune(guess[i]), Score: Hit}
		} else {
			answerCounts[answer[i]]++
		}
	}

	// Second pass: identify presents
	for i := 0; i < 5; i++ {
		if res[i].Score == Hit {
			continue
		}
		if answerCounts[guess[i]] > 0 {
			res[i] = LetterResult{Char: rune(guess[i]), Score: Present}
			answerCounts[guess[i]]--
		} else {
			res[i] = LetterResult{Char: rune(guess[i]), Score: Miss}
		}
	}

	// Update letterStates
	for _, r := range res {
		existing, ok := letterStates[unicode.ToUpper(r.Char)]
		if !ok || r.Score > existing {
			letterStates[unicode.ToUpper(r.Char)] = r.Score
		}
	}
	return res
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
	answer := pickAnswer()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to Wordle CLI!")

	letterStates := make(map[rune]Score)
	for r := 'A'; r <= 'Z'; r++ {
		letterStates[r] = Unknown
	}

	var history [][]LetterResult

	for round := 1; round <= maxRounds; round++ {
		fmt.Printf("\nRound %d/%d\n", round, maxRounds)

		for _, past := range history {
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

		result := scoreGuess(answer, guess, letterStates)
		history = append(history, result)

		correct := true
		for _, r := range result {
			if r.Score != Hit {
				correct = false
				break
			}
		}
		if correct {
			fmt.Println("Congratulations! You guessed the word.")
			return
		}
	}
	fmt.Printf("Game over! The word was: %s\n", answer)
}
