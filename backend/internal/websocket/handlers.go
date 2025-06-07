package websocket

import (
	"encoding/json"
	"log"
	"time"

	"game2048/pkg/models"

	"github.com/google/uuid"
)

// handleMove handles move requests from clients
func (c *Client) handleMove(data interface{}) {
	// Parse move request
	dataBytes, err := json.Marshal(data)
	if err != nil {
		c.sendError("Invalid move data")
		return
	}

	var moveRequest models.MoveRequest
	if err := json.Unmarshal(dataBytes, &moveRequest); err != nil {
		c.sendError("Invalid move request format")
		return
	}

	// Validate direction
	if moveRequest.Direction != models.DirectionUp &&
		moveRequest.Direction != models.DirectionDown &&
		moveRequest.Direction != models.DirectionLeft &&
		moveRequest.Direction != models.DirectionRight {
		c.sendError("Invalid direction")
		return
	}

	// Get current game state
	gameState, err := c.getCurrentGameState()
	if err != nil {
		c.sendError("Failed to get game state")
		return
	}

	if gameState == nil {
		c.sendError("No active game found. Start a new game first.")
		return
	}

	// Check if game is already over
	if gameState.GameOver || gameState.Victory {
		c.sendError("Game is already finished")
		return
	}

	// Execute move based on game mode
	var newBoard models.Board
	var scoreGained int
	var moved bool

	if gameState.GameMode == models.GameModeChallenge && gameState.DisabledCell != nil {
		newBoard, scoreGained, moved = c.hub.gameEngine.MoveWithDisabledCell(gameState.Board, moveRequest.Direction, gameState.DisabledCell)
	} else {
		newBoard, scoreGained, moved = c.hub.gameEngine.Move(gameState.Board, moveRequest.Direction)
	}

	if !moved {
		c.sendError("Invalid move - no tiles moved")
		return
	}

	// Update game state
	gameState.Board = newBoard
	gameState.Score += scoreGained

	// Check for victory
	if c.hub.gameEngine.IsVictory(gameState.Board) {
		gameState.Victory = true
	}

	// Check for game over (use appropriate logic based on game mode)
	if gameState.GameMode == models.GameModeChallenge && gameState.DisabledCell != nil {
		if c.hub.gameEngine.IsGameOverWithDisabledCell(gameState.Board, gameState.DisabledCell) {
			gameState.GameOver = true
		}
	} else {
		if c.hub.gameEngine.IsGameOver(gameState.Board) {
			gameState.GameOver = true
		}
	}

	// Save updated game state to cache
	if c.hub.cache != nil {
		// Cache for 1 hour
		if err := c.hub.cache.SetGameSession(c.userID, gameState, time.Hour); err != nil {
			log.Printf("Failed to cache game session: %v", err)
		}
	}

	// Only save to database if game is finished (for leaderboard purposes)
	if gameState.GameOver || gameState.Victory {
		// Try to update first, if it fails (game not in DB), create it
		if err := c.hub.db.UpdateGame(gameState); err != nil {
			log.Printf("Failed to update game state, trying to create: %v", err)
			// Game doesn't exist in database, create it
			if err := c.hub.db.CreateGame(gameState); err != nil {
				log.Printf("Failed to create game state in database: %v", err)
				// Don't return error here, game state is still cached
			} else {
				log.Printf("Successfully created finished game in database")
			}
		} else {
			log.Printf("Successfully updated finished game in database")
		}
	}

	// Send response
	response := models.GameResponse{
		Board:        gameState.Board,
		Score:        gameState.Score,
		GameOver:     gameState.GameOver,
		Victory:      gameState.Victory,
		GameMode:     gameState.GameMode,
		DisabledCell: gameState.DisabledCell,
	}

	if gameState.Victory {
		response.Message = "Congratulations! You merged two 8192 tiles and won!"
	} else if gameState.GameOver {
		response.Message = "Game Over! No more moves available."
	}

	message := models.WebSocketMessage{
		Type: "game_state",
		Data: response,
	}

	c.sendMessage(message)

	// If game is finished, update leaderboards
	if gameState.GameOver || gameState.Victory {
		go c.updateLeaderboards(gameState)
	}
}

// handleNewGame handles new game requests
func (c *Client) handleNewGame(data interface{}) {
	// Parse new game request
	dataBytes, err := json.Marshal(data)
	if err != nil {
		c.sendError("Invalid new game data")
		return
	}

	var newGameRequest models.NewGameRequest
	if err := json.Unmarshal(dataBytes, &newGameRequest); err != nil {
		// Default to classic mode if no mode specified
		newGameRequest.GameMode = models.GameModeClassic
	}

	// Validate game mode
	if newGameRequest.GameMode != models.GameModeClassic && newGameRequest.GameMode != models.GameModeChallenge {
		newGameRequest.GameMode = models.GameModeClassic
	}

	// Create new game based on mode
	var board models.Board
	var disabledCell *models.DisabledCell

	if newGameRequest.GameMode == models.GameModeChallenge {
		board, disabledCell = c.hub.gameEngine.NewGameWithMode(models.GameModeChallenge)
	} else {
		board = c.hub.gameEngine.NewGame()
	}

	gameID := uuid.New()

	gameState := &models.GameState{
		ID:           gameID,
		UserID:       c.userID,
		Board:        board,
		Score:        0,
		GameOver:     false,
		Victory:      false,
		GameMode:     newGameRequest.GameMode,
		DisabledCell: disabledCell,
	}

	// Save new game state to cache
	if c.hub.cache != nil {
		// Cache for 1 hour
		if err := c.hub.cache.SetGameSession(c.userID, gameState, time.Hour); err != nil {
			log.Printf("Failed to cache new game session: %v", err)
			c.sendError("Failed to create new game")
			return
		}
	} else {
		// Fallback to database if no cache
		if err := c.hub.db.CreateGame(gameState); err != nil {
			log.Printf("Failed to create new game: %v", err)
			c.sendError("Failed to create new game")
			return
		}
	}

	// Update client's game ID
	c.gameID = gameID

	// Send response
	statusMessage := "New game started!"
	if gameState.GameMode == models.GameModeChallenge {
		statusMessage = "Challenge mode started! One cell is disabled."
	}

	response := models.GameResponse{
		Board:        gameState.Board,
		Score:        gameState.Score,
		GameOver:     gameState.GameOver,
		Victory:      gameState.Victory,
		GameMode:     gameState.GameMode,
		DisabledCell: gameState.DisabledCell,
		Message:      statusMessage,
	}

	message := models.WebSocketMessage{
		Type: "game_state",
		Data: response,
	}

	c.sendMessage(message)
}

// handleGetLeaderboard handles leaderboard requests
func (c *Client) handleGetLeaderboard(data interface{}) {
	// Parse leaderboard request
	dataBytes, err := json.Marshal(data)
	if err != nil {
		c.sendError("Invalid leaderboard data")
		return
	}

	var leaderboardRequest models.LeaderboardRequest
	if err := json.Unmarshal(dataBytes, &leaderboardRequest); err != nil {
		c.sendError("Invalid leaderboard request format")
		return
	}

	// Validate leaderboard type
	if leaderboardRequest.Type != models.LeaderboardDaily &&
		leaderboardRequest.Type != models.LeaderboardWeekly &&
		leaderboardRequest.Type != models.LeaderboardMonthly &&
		leaderboardRequest.Type != models.LeaderboardAll {
		c.sendError("Invalid leaderboard type")
		return
	}

	// Validate game mode
	if leaderboardRequest.GameMode != models.GameModeClassic &&
		leaderboardRequest.GameMode != models.GameModeChallenge {
		leaderboardRequest.GameMode = models.GameModeClassic
	}

	// Get leaderboard entries for the specific game mode
	entries, err := c.hub.db.GetLeaderboardByMode(leaderboardRequest.Type, leaderboardRequest.GameMode, 100)
	if err != nil {
		log.Printf("Failed to get leaderboard: %v", err)
		c.sendError("Failed to get leaderboard")
		return
	}

	// Send response
	response := models.LeaderboardResponse{
		Type:     leaderboardRequest.Type,
		Rankings: entries,
	}

	message := models.WebSocketMessage{
		Type: "leaderboard",
		Data: response,
	}

	c.sendMessage(message)
}

// getCurrentGameState gets the current game state for the client
func (c *Client) getCurrentGameState() (*models.GameState, error) {
	// Try to get from Redis cache first
	if c.hub.cache != nil {
		gameState, err := c.hub.cache.GetGameSession(c.userID)
		if err == nil && gameState != nil {
			c.gameID = gameState.ID
			return gameState, nil
		}
	}

	// Cache miss or no cache, try database as fallback
	if c.gameID == uuid.Nil {
		// Try to get user's active game
		gameState, err := c.hub.db.GetUserActiveGame(c.userID)
		if err != nil {
			return nil, err
		}
		if gameState != nil {
			c.gameID = gameState.ID
			// Cache the game state
			if c.hub.cache != nil {
				if err := c.hub.cache.SetGameSession(c.userID, gameState, time.Hour); err != nil {
					log.Printf("Failed to cache game session: %v", err)
				}
			}
		}
		return gameState, nil
	}

	// Get game by ID from database
	gameState, err := c.hub.db.GetGame(c.gameID.String(), c.userID)
	if err == nil && gameState != nil && c.hub.cache != nil {
		// Cache the game state
		if err := c.hub.cache.SetGameSession(c.userID, gameState, time.Hour); err != nil {
			log.Printf("Failed to cache game session: %v", err)
		}
	}
	return gameState, err
}

// updateLeaderboards updates the leaderboard cache when a game finishes
func (c *Client) updateLeaderboards(gameState *models.GameState) {
	log.Printf("Game finished for user %s with score %d", c.userID, gameState.Score)

	// Invalidate leaderboard caches so they will be refreshed on next request
	if c.hub.cache != nil {
		leaderboardTypes := []models.LeaderboardType{
			models.LeaderboardDaily,
			models.LeaderboardWeekly,
			models.LeaderboardMonthly,
			models.LeaderboardAll,
		}

		for _, lbType := range leaderboardTypes {
			if err := c.hub.cache.InvalidateLeaderboard(lbType); err != nil {
				log.Printf("Failed to invalidate %s leaderboard cache: %v", lbType, err)
			} else {
				log.Printf("Invalidated %s leaderboard cache", lbType)
			}
		}
	}

	// Optionally broadcast leaderboard updates to connected clients
	// This could be expensive with many concurrent games, so we'll skip it for now
	// go c.hub.broadcastLeaderboardUpdate(models.LeaderboardAll)
}

// broadcastLeaderboardUpdate broadcasts leaderboard updates to all connected clients
func (h *Hub) broadcastLeaderboardUpdate(leaderboardType models.LeaderboardType) {
	entries, err := h.db.GetLeaderboard(leaderboardType, 100)
	if err != nil {
		log.Printf("Failed to get leaderboard for broadcast: %v", err)
		return
	}

	response := models.LeaderboardResponse{
		Type:     leaderboardType,
		Rankings: entries,
	}

	message := models.WebSocketMessage{
		Type: "leaderboard_update",
		Data: response,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal leaderboard update: %v", err)
		return
	}

	// Broadcast to all clients
	h.broadcast <- data
}
