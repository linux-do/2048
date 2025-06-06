package database

import (
	"fmt"
	"log"
	"time"

	"game2048/pkg/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormDB wraps the GORM database connection and implements Database interface
type GormDB struct {
	db *gorm.DB
}

// Ensure GormDB implements Database interface
var _ Database = (*GormDB)(nil)

// NewGormDB creates a new GORM database connection
func NewGormDB(host, port, user, password, dbname, sslmode string) (*GormDB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database with GORM")

	gormDB := &GormDB{db: db}

	// Auto-migrate the schema
	if err := gormDB.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return gormDB, nil
}

// AutoMigrate runs database migrations
func (g *GormDB) AutoMigrate() error {
	return g.db.AutoMigrate(
		&models.GormUser{},
		&models.GormGame{},
		&models.GormDailyLeaderboard{},
		&models.GormWeeklyLeaderboard{},
		&models.GormMonthlyLeaderboard{},
	)
}

// Close closes the database connection
func (g *GormDB) Close() error {
	sqlDB, err := g.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateUser creates a new user
func (g *GormDB) CreateUser(user *models.User) error {
	gormUser := &models.GormUser{}
	gormUser.FromUser(user)

	// Use GORM's upsert functionality
	result := g.db.Where("provider = ? AND provider_id = ?", user.Provider, user.ProviderID).
		Assign(models.GormUser{
			Email:     user.Email,
			Name:      user.Name,
			Avatar:    user.Avatar,
			UpdatedAt: time.Now(),
		}).
		FirstOrCreate(gormUser)

	if result.Error != nil {
		return fmt.Errorf("failed to create user: %w", result.Error)
	}

	// Update the original user with the database values
	*user = *gormUser.ToUser()
	return nil
}

// GetUser retrieves a user by ID
func (g *GormDB) GetUser(userID string) (*models.User, error) {
	var gormUser models.GormUser
	result := g.db.Where("id = ?", userID).First(&gormUser)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}

	return gormUser.ToUser(), nil
}

// GetUserByProvider retrieves a user by provider and provider ID
func (g *GormDB) GetUserByProvider(provider, providerID string) (*models.User, error) {
	var gormUser models.GormUser
	result := g.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&gormUser)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by provider: %w", result.Error)
	}

	return gormUser.ToUser(), nil
}

// CreateGame creates a new game
func (g *GormDB) CreateGame(game *models.GameState) error {
	gormGame := &models.GormGame{}
	gormGame.FromGameState(game)

	result := g.db.Create(gormGame)
	if result.Error != nil {
		return fmt.Errorf("failed to create game: %w", result.Error)
	}

	// Update the original game with the database values
	*game = *gormGame.ToGameState()
	return nil
}

// UpdateGame updates an existing game
func (g *GormDB) UpdateGame(game *models.GameState) error {
	gormGame := &models.GormGame{}
	gormGame.FromGameState(game)

	result := g.db.Model(&models.GormGame{}).
		Where("id = ? AND user_id = ?", game.ID, game.UserID).
		Updates(map[string]interface{}{
			"board":      gormGame.Board,
			"score":      game.Score,
			"game_over":  game.GameOver,
			"victory":    game.Victory,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update game: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("game not found or not owned by user")
	}

	game.UpdatedAt = time.Now()
	return nil
}

// GetGame retrieves a game by ID and user ID
func (g *GormDB) GetGame(gameID, userID string) (*models.GameState, error) {
	var gormGame models.GormGame
	result := g.db.Where("id = ? AND user_id = ?", gameID, userID).First(&gormGame)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("failed to get game: %w", result.Error)
	}

	return gormGame.ToGameState(), nil
}

// GetUserActiveGame retrieves the user's active (non-finished) game
func (g *GormDB) GetUserActiveGame(userID string) (*models.GameState, error) {
	var gormGame models.GormGame
	result := g.db.Where("user_id = ? AND game_over = ? AND victory = ?", userID, false, false).
		Order("updated_at DESC").
		First(&gormGame)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // No active game found
		}
		return nil, fmt.Errorf("failed to get active game: %w", result.Error)
	}

	return gormGame.ToGameState(), nil
}

// GetLeaderboard retrieves leaderboard entries
func (g *GormDB) GetLeaderboard(leaderboardType models.LeaderboardType, limit int) ([]models.LeaderboardEntry, error) {
	var entries []models.GormLeaderboardEntry

	// Build subquery to get max score per user
	subquery := g.db.Table("games").
		Select("user_id, MAX(score) as max_score").
		Where("game_over = ? OR victory = ?", true, true)

	switch leaderboardType {
	case models.LeaderboardDaily:
		subquery = subquery.Where("created_at >= CURRENT_DATE")
	case models.LeaderboardWeekly:
		subquery = subquery.Where("created_at >= DATE_TRUNC('week', CURRENT_DATE)")
	case models.LeaderboardMonthly:
		subquery = subquery.Where("created_at >= DATE_TRUNC('month', CURRENT_DATE)")
	case models.LeaderboardAll:
		// No additional filter for all-time leaderboard
	default:
		return nil, fmt.Errorf("invalid leaderboard type")
	}

	subquery = subquery.Group("user_id")

	// Main query to get full game details for the max score games
	query := g.db.Table("games g").
		Select("g.user_id, u.name as user_name, u.avatar as user_avatar, g.score, g.id as game_id, g.created_at, ROW_NUMBER() OVER (ORDER BY g.score DESC) as rank").
		Joins("JOIN users u ON g.user_id = u.id").
		Joins("JOIN (?) max_scores ON g.user_id = max_scores.user_id AND g.score = max_scores.max_score", subquery).
		Where("g.game_over = ? OR g.victory = ?", true, true).
		Order("g.score DESC").
		Limit(limit)

	result := query.Scan(&entries)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", result.Error)
	}

	// Convert to regular LeaderboardEntry
	var leaderboardEntries []models.LeaderboardEntry
	for _, entry := range entries {
		leaderboardEntries = append(leaderboardEntries, *entry.ToLeaderboardEntry())
	}

	return leaderboardEntries, nil
}

// GetDB returns the underlying GORM database instance
func (g *GormDB) GetDB() *gorm.DB {
	return g.db
}
