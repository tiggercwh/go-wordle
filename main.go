package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

const maxRounds = 6

var wordList = []string{
	"hello", "world", "quite", "fancy", "fresh",
	"panic", "crazy", "buggy",
}

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

type FeedbackKey struct {
	GroupResult [5]int
	Hits        int
	Presents    int
}

func scoreGuess(answer, guess string) []LetterResult {
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

	return res
}

func summarizeResult(result []LetterResult) FeedbackKey {
	var key FeedbackKey
	for i, r := range result {
		switch r.Score {
		case Hit:
			key.GroupResult[i] = 2
			key.Hits++
		case Present:
			key.GroupResult[i] = 1
			key.Presents++
		case Miss:
			key.GroupResult[i] = 0
		}
	}
	return key
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
	fmt.Println("Welcome to Wordle CLI!")

	candidates := make([]string, len(wordList))
	copy(candidates, wordList)

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

		// Group words with same feedback
		grouped := make(map[FeedbackKey][]string)
		resultMap := make(map[FeedbackKey][]LetterResult)

		for _, cand := range candidates {
			result := scoreGuess(cand, guess)
			fb := summarizeResult(result)
			grouped[fb] = append(grouped[fb], cand)
			resultMap[fb] = result
		}

		// Choose least helpful feedback
		var bestFB FeedbackKey
		bestScore := [2]int{6, 6}

		for fb := range grouped {
			if fb.Hits < bestScore[0] || (fb.Hits == bestScore[0] && fb.Presents < bestScore[1]) {
				bestFB = fb
				bestScore = [2]int{fb.Hits, fb.Presents}
			}
		}

		candidates = grouped[bestFB]
		result := resultMap[bestFB]
		history = append(history, result)
		printGuessResult(result)

		if len(candidates) == 1 && guess == candidates[0] {
			fmt.Println("Congratulations! You guessed the word.")
			return
		}
	}

	fmt.Printf("Game over! The word was: %s\n", candidates[0])
}
