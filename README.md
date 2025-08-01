# Go Wordle - Server-Client Architecture

A Wordle game implementation with server-client architecture, featuring true Absurdle behavior where the answer dynamically changes based on player guesses.

## Architecture Overview

### Server (`server/main.go`)

- **Game Logic**: All game state management and Absurdle logic runs on the server
- **API Endpoints**: RESTful HTTP endpoints for game operations
- **State Management**: In-memory game state with thread-safe access using `sync.RWMutex`

### Client (`client/main.go`)

- **CLI Interface**: Minimal terminal UI with coloring
- **HTTP Communication**: Communicates with server via REST API calls
- **Error Handling**: Graceful handling of network errors and server responses

### Shared Types (`gameModel/types.go`)

- **Type Safety**: Common data structures shared between client and server
- **JSON Serialization**: Consistent API contract between components
- **Maintainability**: Single source of truth for all game-related types

## Setup & Testing

### Prerequisites

- Go 1.24.5 or later

### Quick Start

1. **Install Dependencies**

   ```bash
   go mod tidy
   ```

2. **Start Server**

   ```bash
   cd server
   # Basic usage with default word list
   go run main.go

   # Or use a custom word list (CSV format, one word per line)
   # Only 5-letter words will be used
   go run main.go -wordlist=../wordlist.csv
   ```

   Server starts on `http://localhost:8080`

   > **Note**: The word list should be a CSV file with one word per line. Only 5-letter words will be used. An example `wordlist.csv` is provided in the project root.

3. **Run Client** (in new terminal)
   ```bash
   cd client
   go run main.go
   ```

### Testing the API

```bash
# Create new game
curl -X POST http://localhost:8080/api/game/new

# Submit guess
curl -X POST http://localhost:8080/api/game/GAME_ID/guess \
  -H "Content-Type: application/json" \
  -d '{"word": "hello"}'

# Get game state
curl http://localhost:8080/api/game/GAME_ID
```

## Development Process & Considerations

### 1. Task sequence and priority

**Priority**: 1 -> 3 -> 2 -> 4
I ordered the priority this way because I think 3 is independent from 2 and 4 in the sense that it is still a client-side only version. After that I believe that it is ideal to have task 2 being setup if I have the time to implement task 4, as the server-client structure will allow us to add multi-player feature relatively easier.

### 2. Terminal as Client (Task 1)

I have chosen the terminal as the client because I think there isn't an obvious need initially to develop a separate frontend like a webpage, it is also easier to manage when I need to extend to other tasks as the code can be shared fairly easy. It also doesn't kill the possibility to create a more sophisticated frontend later if we find the need later.

### 3. Testing (Task 3)

Task 3 is the point where I want to safeguard the behaviour with test here, because the logic is not as straight forward as the original wordle, and it is not easy to realise undesired results. I didn't implement the test immediately just because I want to delay refactors until task 2, and it would be better to write the test once things get a little bit stabler.(Especially after I have written the shared types)

### 4. Server implementation (Task 2)

The main concern/objective when doing this task is to implement in a fairly flexible/scalable way so that it allows extension to task 4 (although we have not done it yet). It is always tricky to take the balance here especially in situations when initiative can change, if possible I do not want to develop in a way that commit too much in certain setup. (For example if I think that multiplayer is a must, I might have written websocket-related code in advance)
I have also done some refactor here to share types and do the client and server separation.

## Code Structure & Comments

### Key Functions in Server

#### `scoreGuess(answer, guess string) []LetterResult`

Implements Wordle scoring algorithm

#### `summarizeResult(result []LetterResult) FeedbackKey`

Converts letter results and group candidate words by their feedback patterns

#### `handleGuess()`

Core Absurdle Logic

1. Group all remaining candidates by their feedback patterns
2. Choose the least helpful feedback (fewest hits + presents)
3. Update candidate list to only include words matching that feedback
4. If only one candidate remains, lock it in for final guess

## Possible Future Enhancements

- **Multiplayer Support**
- **Database Persistence**
- **User Authentication**
- **Rate Limiting**
- **WebSocket Support**
