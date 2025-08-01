package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/tiggercwh/go-wordle/gameModel"
)

const maxRounds = 6

var (
	wordList     []string
	wordListPath = flag.String("wordlist", "", "Path to a CSV file containing words (one per line)")
)

type GameServer struct {
	games map[string]*gameModel.GameState
	mutex sync.RWMutex
}

func NewGameServer() *GameServer {
	return &GameServer{
		games: make(map[string]*gameModel.GameState),
	}
}

func (gs *GameServer) createGame() *gameModel.GameState {
	game := &gameModel.GameState{
		ID:           generateGameID(),
		Round:        0,
		MaxRounds:    maxRounds,
		History:      make([][]gameModel.LetterResult, 0),
		Candidates:   make([]string, len(wordList)),
		GameOver:     false,
		Won:          false,
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
	}
	copy(game.Candidates, wordList)
	fmt.Println("Created game with candidates: ", game.Candidates)
	gs.mutex.Lock()
	gs.games[game.ID] = game
	gs.mutex.Unlock()
	return game
}

func (gs *GameServer) getGame(gameID string) (*gameModel.GameState, bool) {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()
	game, exists := gs.games[gameID]
	return game, exists
}

func (gs *GameServer) updateGame(gameID string, game *gameModel.GameState) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()
	game.LastActivity = time.Now().Format(time.RFC3339)
	gs.games[gameID] = game
}

func generateGameID() string {
	return fmt.Sprintf("game_%d", time.Now().UnixNano())
}

func scoreGuess(answer, guess string) []gameModel.LetterResult {
	res := make([]gameModel.LetterResult, 5)
	answerCounts := make(map[byte]int)
	for i := 0; i < 5; i++ {
		if guess[i] == answer[i] {
			res[i] = gameModel.LetterResult{Char: rune(guess[i]), Score: gameModel.Hit}
		} else {
			answerCounts[answer[i]]++
		}
	}
	for i := 0; i < 5; i++ {
		if res[i].Score == gameModel.Hit {
			continue
		}
		if answerCounts[guess[i]] > 0 {
			res[i] = gameModel.LetterResult{Char: rune(guess[i]), Score: gameModel.Present}
			answerCounts[guess[i]]--
		} else {
			res[i] = gameModel.LetterResult{Char: rune(guess[i]), Score: gameModel.Miss}
		}
	}
	return res
}

func summarizeResult(result []gameModel.LetterResult) gameModel.FeedbackKey {
	var key gameModel.FeedbackKey
	for i, r := range result {
		switch r.Score {
		case gameModel.Hit:
			key.GroupResult[i] = 2
			key.Hits++
		case gameModel.Present:
			key.GroupResult[i] = 1
			key.Presents++
		case gameModel.Miss:
			key.GroupResult[i] = 0
		}
	}
	return key
}

func (gs *GameServer) handleNewGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	game := gs.createGame()
	response := gameModel.NewGameResponse{
		Success:   true,
		Message:   "New game created successfully",
		GameState: *game,
	}
	json.NewEncoder(w).Encode(response)
}

func (gs *GameServer) handleGuess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req gameModel.GuessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	vars := mux.Vars(r)
	gameID := vars["gameID"]
	game, exists := gs.getGame(gameID)
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if game.GameOver {
		response := gameModel.GuessResponse{
			Success:  false,
			Message:  "Game is already over",
			GameOver: true,
			Won:      game.Won,
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	if len(req.Word) != 5 {
		response := gameModel.GuessResponse{
			Success: false,
			Message: "Please enter a valid 5-letter word",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	guess := req.Word
	for i := range guess {
		if guess[i] >= 'A' && guess[i] <= 'Z' {
			guess = guess[:i] + string(guess[i]+32) + guess[i+1:]
		}
	}
	game.Round++
	grouped := make(map[gameModel.FeedbackKey][]string)
	resultMap := make(map[gameModel.FeedbackKey][]gameModel.LetterResult)
	for _, cand := range game.Candidates {
		result := scoreGuess(cand, guess)
		fb := summarizeResult(result)
		grouped[fb] = append(grouped[fb], cand)
		resultMap[fb] = result
	}
	var bestFB gameModel.FeedbackKey
	bestScore := [2]int{6, 6}
	for fb := range grouped {
		// We use <= for Presents just to match the result of example 2 provided
		if fb.Hits < bestScore[0] || (fb.Hits == bestScore[0] && fb.Presents <= bestScore[1]) {
			bestFB = fb
			bestScore = [2]int{fb.Hits, fb.Presents}
		}
	}
	game.Candidates = grouped[bestFB]
	result := resultMap[bestFB]
	game.History = append(game.History, result)
	game.GameOver = game.Round >= maxRounds
	if len(game.Candidates) == 1 && guess == game.Candidates[0] {
		game.Won = true
		game.GameOver = true
	}
	gs.updateGame(gameID, game)
	response := gameModel.GuessResponse{
		Success:   true,
		Message:   "Guess processed successfully",
		Result:    result,
		GameState: game,
		GameOver:  game.GameOver,
		Won:       game.Won,
	}
	json.NewEncoder(w).Encode(response)
}

func (gs *GameServer) handleGetGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	vars := mux.Vars(r)
	gameID := vars["gameID"]
	game, exists := gs.getGame(gameID)
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(game)
}

func loadWordList() error {
	// Default word list (all 5 letters)
	defaultWords := []string{
		"hello", "world", "quite", "fancy", "fresh",
		"panic", "crazy", "buggy",
	}

	if *wordListPath == "" {
		wordList = defaultWords
		return nil
	}

	file, err := os.Open(*wordListPath)
	if err != nil {
		return fmt.Errorf("failed to open word list file: %w", err)
	}
	defer file.Close()

	// Use a map to track unique words
	uniqueWords := make(map[string]bool)

	// Add default words to ensure we always have some valid words
	for _, word := range defaultWords {
		uniqueWords[word] = true
	}

	// Read words from CSV
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading CSV: %w", err)
		}
		for _, word := range record {
			word = strings.TrimSpace(strings.ToLower(word))
			// Only include 5-letter words
			if len(word) == 5 {
				uniqueWords[word] = true
			}
		}
	}

	// Convert map keys to slice
	wordList = make([]string, 0, len(uniqueWords))
	for word := range uniqueWords {
		wordList = append(wordList, word)
	}

	if len(wordList) == 0 {
		return fmt.Errorf("no valid 5-letter words found in the word list")
	}

	log.Printf("Loaded %d unique 5-letter words", len(wordList))

	return nil
}

func main() {
	flag.Parse()

	if err := loadWordList(); err != nil {
		log.Fatalf("Failed to load word list: %v", err)
	}

	r := mux.NewRouter()
	gameServer := NewGameServer()

	r.HandleFunc("/api/game/new", gameServer.handleNewGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/game/{gameID}/guess", gameServer.handleGuess).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/game/{gameID}", gameServer.handleGetGame).Methods("GET")

	log.Printf("Server starting with %d words loaded", len(wordList))
	log.Fatal(http.ListenAndServe(":8080", r))
}
