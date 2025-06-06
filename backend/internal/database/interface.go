package database

import "game2048/pkg/models"

// Database defines the interface for database operations
type Database interface {
	// User operations
	CreateUser(user *models.User) error
	GetUser(userID string) (*models.User, error)
	GetUserByProvider(provider, providerID string) (*models.User, error)

	// Game operations
	CreateGame(game *models.GameState) error
	UpdateGame(game *models.GameState) error
	GetGame(gameID, userID string) (*models.GameState, error)
	GetUserActiveGame(userID string) (*models.GameState, error)

	// Leaderboard operations
	GetLeaderboard(leaderboardType models.LeaderboardType, limit int) ([]models.LeaderboardEntry, error)

	// Connection management
	Close() error
}
