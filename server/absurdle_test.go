package main

import (
	"testing"

	"github.com/tiggercwh/go-wordle/gameModel"
)

// TestAbsurdleExample1 tests the first example from the specification
func TestAbsurdleExample1(t *testing.T) {
	server := NewGameServer()

	// Create a game with the specified word list
	wordList = []string{"hello", "world", "quite", "fancy", "fresh", "panic", "crazy", "buggy"}
	game := server.createGame()

	// Verify initial state
	if len(game.Candidates) != 8 {
		t.Errorf("Expected 8 candidates, got %d", len(game.Candidates))
	}

	testGuess(server, game.ID, "hello", t)
	expectedCandidates := []string{"fancy", "panic", "crazy", "buggy"}
	verifyCandidates(game.ID, expectedCandidates, server, t)

	testGuess(server, game.ID, "world", t)
	expectedCandidates = []string{"fancy", "panic", "buggy"}
	verifyCandidates(game.ID, expectedCandidates, server, t)

	testGuess(server, game.ID, "fresh", t)
	expectedCandidates = []string{"panic", "buggy"}
	verifyCandidates(game.ID, expectedCandidates, server, t)

	testGuess(server, game.ID, "crazy", t)
	expectedCandidates = []string{"panic"}
	verifyCandidates(game.ID, expectedCandidates, server, t)

	testGuess(server, game.ID, "quite", t)

	testGuess(server, game.ID, "fancy", t)

	game, _ = server.getGame(game.ID)
	if !game.GameOver {
		t.Error("Expected game to be over after 6 rounds")
	}
}

// Helper functions for testing
func testGuess(server *GameServer, gameID string, word string, t *testing.T) []gameModel.LetterResult {
	// Simulate the guess processing logic
	game, exists := server.getGame(gameID)
	if !exists {
		t.Fatalf("Game %s not found", gameID)
	}

	// Group words with same feedback
	grouped := make(map[gameModel.FeedbackKey][]string)
	resultMap := make(map[gameModel.FeedbackKey][]gameModel.LetterResult)

	for _, cand := range game.Candidates {
		result := scoreGuess(cand, word)
		fb := summarizeResult(result)
		grouped[fb] = append(grouped[fb], cand)
		resultMap[fb] = result
	}

	// Choose least helpful feedback
	var bestFB gameModel.FeedbackKey
	bestScore := [2]int{6, 6}

	for fb := range grouped {
		if fb.Hits < bestScore[0] || (fb.Hits == bestScore[0] && fb.Presents < bestScore[1]) {
			bestFB = fb
			bestScore = [2]int{fb.Hits, fb.Presents}
		}
	}

	// Update game state
	game.Round++
	game.Candidates = grouped[bestFB]
	result := resultMap[bestFB]
	game.History = append(game.History, result)

	// Check game over conditions
	game.GameOver = game.Round >= game.MaxRounds
	if len(game.Candidates) == 1 && word == game.Candidates[0] {
		game.Won = true
		game.GameOver = true
	}

	server.updateGame(gameID, game)
	return result
}

func verifyCandidates(gameID string, expected []string, server *GameServer, t *testing.T) {
	game, _ := server.getGame(gameID)
	if len(game.Candidates) != len(expected) {
		t.Errorf("Expected %d candidates, got %d", len(expected), len(game.Candidates))
		return
	}

	// Create a map for easy comparison
	expectedMap := make(map[string]bool)
	for _, word := range expected {
		expectedMap[word] = true
	}

	for _, candidate := range game.Candidates {
		if !expectedMap[candidate] {
			t.Errorf("Unexpected candidate: %s", candidate)
		}
	}
}
