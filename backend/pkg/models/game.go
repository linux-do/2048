package models

import (
	"time"

	"github.com/google/uuid"
)

// Direction represents the direction of a move
type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

// GameState represents the current state of a 2048 game
type GameState struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Board     Board     `json:"board" db:"board"`
	Score     int       `json:"score" db:"score"`
	GameOver  bool      `json:"game_over" db:"game_over"`
	Victory   bool      `json:"victory" db:"victory"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Board represents a 4x4 game board
type Board [4][4]int

// User represents a user in the system
type User struct {
	ID         string    `json:"id" db:"id"`
	Email      string    `json:"email" db:"email"`
	Name       string    `json:"name" db:"name"`
	Avatar     string    `json:"avatar" db:"avatar"`
	Provider   string    `json:"provider" db:"provider"`
	ProviderID string    `json:"provider_id" db:"provider_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// LeaderboardEntry represents an entry in the leaderboard
type LeaderboardEntry struct {
	UserID     string    `json:"user_id" db:"user_id"`
	UserName   string    `json:"user_name" db:"user_name"`
	UserAvatar string    `json:"user_avatar" db:"user_avatar"`
	Score      int       `json:"score" db:"score"`
	Rank       int       `json:"rank" db:"rank"`
	GameID     uuid.UUID `json:"game_id" db:"game_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// LeaderboardType represents different types of leaderboards
type LeaderboardType string

const (
	LeaderboardDaily   LeaderboardType = "daily"
	LeaderboardWeekly  LeaderboardType = "weekly"
	LeaderboardMonthly LeaderboardType = "monthly"
	LeaderboardAll     LeaderboardType = "all"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// MoveRequest represents a move request from the client
type MoveRequest struct {
	Direction Direction `json:"direction"`
}

// NewGameRequest represents a new game request
type NewGameRequest struct {
	// Empty for now, can be extended with game options
}

// LeaderboardRequest represents a leaderboard request
type LeaderboardRequest struct {
	Type LeaderboardType `json:"type"`
}

// GameResponse represents the response sent to client after a move
type GameResponse struct {
	Board    Board  `json:"board"`
	Score    int    `json:"score"`
	GameOver bool   `json:"game_over"`
	Victory  bool   `json:"victory"`
	Message  string `json:"message,omitempty"`
}

// LeaderboardResponse represents the leaderboard response
type LeaderboardResponse struct {
	Type     LeaderboardType    `json:"type"`
	Rankings []LeaderboardEntry `json:"rankings"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// Constants for the game
const (
	BoardSize    = 4
	VictoryTile  = 16384 // Two 8192 tiles merged
	InitialTiles = 2
)

// NewBoard creates a new empty board
func NewBoard() Board {
	return Board{}
}

// IsEmpty checks if a cell is empty
func (b *Board) IsEmpty(row, col int) bool {
	return b[row][col] == 0
}

// GetEmptyCells returns all empty cell positions
func (b *Board) GetEmptyCells() [][2]int {
	var empty [][2]int
	for i := 0; i < BoardSize; i++ {
		for j := 0; j < BoardSize; j++ {
			if b.IsEmpty(i, j) {
				empty = append(empty, [2]int{i, j})
			}
		}
	}
	return empty
}

// SetCell sets a value at the given position
func (b *Board) SetCell(row, col, value int) {
	if row >= 0 && row < BoardSize && col >= 0 && col < BoardSize {
		b[row][col] = value
	}
}

// GetCell gets the value at the given position
func (b *Board) GetCell(row, col int) int {
	if row >= 0 && row < BoardSize && col >= 0 && col < BoardSize {
		return b[row][col]
	}
	return 0
}

// HasVictoryTile checks if the board contains the victory tile
func (b *Board) HasVictoryTile() bool {
	for i := 0; i < BoardSize; i++ {
		for j := 0; j < BoardSize; j++ {
			if b[i][j] == VictoryTile {
				return true
			}
		}
	}
	return false
}

// IsFull checks if the board is full
func (b *Board) IsFull() bool {
	return len(b.GetEmptyCells()) == 0
}

// Copy creates a deep copy of the board
func (b *Board) Copy() Board {
	var copy Board
	for i := 0; i < BoardSize; i++ {
		for j := 0; j < BoardSize; j++ {
			copy[i][j] = b[i][j]
		}
	}
	return copy
}
