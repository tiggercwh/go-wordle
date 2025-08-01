package gameModel

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
