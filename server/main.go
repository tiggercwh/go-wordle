package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
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
	Char  rune  `json:"char"`
	Score Score `json:"score"`
}

type FeedbackKey struct {
	GroupResult [5]int `json:"groupResult"`
	Hits        int    `json:"hits"`
	Presents    int    `json:"presents"`
}

type GameState struct {
	ID           string           `json:"id"`
	Round        int              `json:"round"`
	MaxRounds    int              `json:"maxRounds"`
	History      [][]LetterResult `json:"history"`
	Candidates   []string         `json:"candidates"`
	GameOver     bool             `json:"gameOver"`
	Won          bool             `json:"won"`
	CreatedAt    time.Time        `json:"createdAt"`
	LastActivity time.Time        `json:"lastActivity"`
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

type GameServer struct {
	games map[string]*GameState
	mutex sync.RWMutex
}

func NewGameServer() *GameServer {
	return &GameServer{
		games: make(map[string]*GameState),
	}
}

func (gs *GameServer) createGame() *GameState {
	game := &GameState{
		ID:           generateGameID(),
		Round:        0,
		MaxRounds:    maxRounds,
		History:      make([][]LetterResult, 0),
		Candidates:   make([]string, len(wordList)),
		GameOver:     false,
		Won:          false,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	copy(game.Candidates, wordList)
	gs.mutex.Lock()
	gs.games[game.ID] = game
	gs.mutex.Unlock()
	return game
}

func (gs *GameServer) getGame(gameID string) (*GameState, bool) {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()
	game, exists := gs.games[gameID]
	return game, exists
}

func (gs *GameServer) updateGame(gameID string, game *GameState) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()
	game.LastActivity = time.Now()
	gs.games[gameID] = game
}

func generateGameID() string {
	return fmt.Sprintf("game_%d", time.Now().UnixNano())
}

func scoreGuess(answer, guess string) []LetterResult {
	res := make([]LetterResult, 5)
	answerCounts := make(map[byte]int)
	for i := 0; i < 5; i++ {
		if guess[i] == answer[i] {
			res[i] = LetterResult{Char: rune(guess[i]), Score: Hit}
		} else {
			answerCounts[answer[i]]++
		}
	}
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
	response := NewGameResponse{
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
	var req GuessRequest
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
		response := GuessResponse{
			Success:  false,
			Message:  "Game is already over",
			GameOver: true,
			Won:      game.Won,
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	if len(req.Word) != 5 {
		response := GuessResponse{
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
	grouped := make(map[FeedbackKey][]string)
	resultMap := make(map[FeedbackKey][]LetterResult)
	for _, cand := range game.Candidates {
		result := scoreGuess(cand, guess)
		fb := summarizeResult(result)
		grouped[fb] = append(grouped[fb], cand)
		resultMap[fb] = result
	}
	var bestFB FeedbackKey
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
	response := GuessResponse{
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

func main() {
	server := NewGameServer()
	router := mux.NewRouter()
	router.HandleFunc("/api/game/new", server.handleNewGame).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/game/{gameID}/guess", server.handleGuess).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/game/{gameID}", server.handleGetGame).Methods("GET")

	fmt.Println("Wordle server listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
