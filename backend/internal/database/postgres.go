package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"game2048/pkg/models"

	_ "github.com/lib/pq"
)

// PostgresDB wraps the database connection and implements Database interface
type PostgresDB struct {
	db *sql.DB
}

// Ensure PostgresDB implements Database interface
var _ Database = (*PostgresDB)(nil)

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(host, port, user, password, dbname, sslmode string) (*PostgresDB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to PostgreSQL database")

	return &PostgresDB{db: db}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// CreateUser creates a new user
func (p *PostgresDB) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (id, email, name, avatar, provider, provider_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (provider, provider_id) 
		DO UPDATE SET 
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar = EXCLUDED.avatar,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	err := p.db.QueryRow(query, user.ID, user.Email, user.Name, user.Avatar,
		user.Provider, user.ProviderID, user.CreatedAt, user.UpdatedAt).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUser retrieves a user by ID
func (p *PostgresDB) GetUser(userID string) (*models.User, error) {
	query := `
		SELECT id, email, name, avatar, provider, provider_id, created_at, updated_at
		FROM users WHERE id = $1`

	user := &models.User{}
	err := p.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByProvider retrieves a user by provider and provider ID
func (p *PostgresDB) GetUserByProvider(provider, providerID string) (*models.User, error) {
	query := `
		SELECT id, email, name, avatar, provider, provider_id, created_at, updated_at
		FROM users WHERE provider = $1 AND provider_id = $2`

	user := &models.User{}
	err := p.db.QueryRow(query, provider, providerID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Avatar,
		&user.Provider, &user.ProviderID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by provider: %w", err)
	}

	return user, nil
}

// CreateGame creates a new game
func (p *PostgresDB) CreateGame(game *models.GameState) error {
	boardJSON, err := json.Marshal(game.Board)
	if err != nil {
		return fmt.Errorf("failed to marshal board: %w", err)
	}

	var disabledCellJSON []byte
	if game.DisabledCell != nil {
		disabledCellJSON, err = json.Marshal(game.DisabledCell)
		if err != nil {
			return fmt.Errorf("failed to marshal disabled cell: %w", err)
		}
	}

	query := `
		INSERT INTO games (id, user_id, board, score, game_over, victory, game_mode, disabled_cell, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	now := time.Now()
	game.CreatedAt = now
	game.UpdatedAt = now

	_, err = p.db.Exec(query, game.ID, game.UserID, boardJSON, game.Score,
		game.GameOver, game.Victory, string(game.GameMode), disabledCellJSON, game.CreatedAt, game.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	return nil
}

// UpdateGame updates an existing game
func (p *PostgresDB) UpdateGame(game *models.GameState) error {
	boardJSON, err := json.Marshal(game.Board)
	if err != nil {
		return fmt.Errorf("failed to marshal board: %w", err)
	}

	var disabledCellJSON []byte
	if game.DisabledCell != nil {
		disabledCellJSON, err = json.Marshal(game.DisabledCell)
		if err != nil {
			return fmt.Errorf("failed to marshal disabled cell: %w", err)
		}
	}

	query := `
		UPDATE games
		SET board = $1, score = $2, game_over = $3, victory = $4, game_mode = $5, disabled_cell = $6, updated_at = $7
		WHERE id = $8 AND user_id = $9`

	game.UpdatedAt = time.Now()

	result, err := p.db.Exec(query, boardJSON, game.Score, game.GameOver,
		game.Victory, string(game.GameMode), disabledCellJSON, game.UpdatedAt, game.ID, game.UserID)

	if err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("game not found or not owned by user")
	}

	return nil
}

// GetGame retrieves a game by ID and user ID
func (p *PostgresDB) GetGame(gameID, userID string) (*models.GameState, error) {
	query := `
		SELECT id, user_id, board, score, game_over, victory, game_mode, disabled_cell, created_at, updated_at
		FROM games WHERE id = $1 AND user_id = $2`

	game := &models.GameState{}
	var boardJSON []byte
	var disabledCellJSON []byte
	var gameMode string

	err := p.db.QueryRow(query, gameID, userID).Scan(
		&game.ID, &game.UserID, &boardJSON, &game.Score,
		&game.GameOver, &game.Victory, &gameMode, &disabledCellJSON, &game.CreatedAt, &game.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	if err := json.Unmarshal(boardJSON, &game.Board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	game.GameMode = models.GameMode(gameMode)

	if len(disabledCellJSON) > 0 {
		var disabledCell models.DisabledCell
		if err := json.Unmarshal(disabledCellJSON, &disabledCell); err != nil {
			return nil, fmt.Errorf("failed to unmarshal disabled cell: %w", err)
		}
		game.DisabledCell = &disabledCell
	}

	return game, nil
}

// GetUserActiveGame retrieves the user's active (non-finished) game
func (p *PostgresDB) GetUserActiveGame(userID string) (*models.GameState, error) {
	query := `
		SELECT id, user_id, board, score, game_over, victory, game_mode, disabled_cell, created_at, updated_at
		FROM games
		WHERE user_id = $1 AND game_over = false AND victory = false
		ORDER BY updated_at DESC
		LIMIT 1`

	game := &models.GameState{}
	var boardJSON []byte
	var disabledCellJSON []byte
	var gameMode string

	err := p.db.QueryRow(query, userID).Scan(
		&game.ID, &game.UserID, &boardJSON, &game.Score,
		&game.GameOver, &game.Victory, &gameMode, &disabledCellJSON, &game.CreatedAt, &game.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active game found
		}
		return nil, fmt.Errorf("failed to get active game: %w", err)
	}

	if err := json.Unmarshal(boardJSON, &game.Board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	game.GameMode = models.GameMode(gameMode)

	if len(disabledCellJSON) > 0 {
		var disabledCell models.DisabledCell
		if err := json.Unmarshal(disabledCellJSON, &disabledCell); err != nil {
			return nil, fmt.Errorf("failed to unmarshal disabled cell: %w", err)
		}
		game.DisabledCell = &disabledCell
	}

	return game, nil
}

// GetLeaderboard retrieves leaderboard entries
func (p *PostgresDB) GetLeaderboard(leaderboardType models.LeaderboardType, limit int) ([]models.LeaderboardEntry, error) {
	var query string
	var args []interface{}

	baseQuery := `
		SELECT
			g.user_id,
			u.name as user_name,
			u.avatar as user_avatar,
			g.score,
			g.id as game_id,
			g.created_at,
			ROW_NUMBER() OVER (ORDER BY g.score DESC) as rank
		FROM (
			SELECT
				user_id,
				MAX(score) as score,
				(ARRAY_AGG(id ORDER BY score DESC))[1] as id,
				(ARRAY_AGG(created_at ORDER BY score DESC))[1] as created_at
			FROM games
			WHERE (game_over = true OR victory = true)`

	var timeFilter string
	switch leaderboardType {
	case models.LeaderboardDaily:
		timeFilter = ` AND created_at >= CURRENT_DATE`
	case models.LeaderboardWeekly:
		timeFilter = ` AND created_at >= DATE_TRUNC('week', CURRENT_DATE)`
	case models.LeaderboardMonthly:
		timeFilter = ` AND created_at >= DATE_TRUNC('month', CURRENT_DATE)`
	case models.LeaderboardAll:
		timeFilter = ""
	default:
		return nil, fmt.Errorf("invalid leaderboard type")
	}

	query = baseQuery + timeFilter + `
			GROUP BY user_id
		) g
		JOIN users u ON g.user_id = u.id
		ORDER BY g.score DESC LIMIT $1`
	args = append(args, limit)

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.UserAvatar,
			&entry.Score, &entry.GameID, &entry.CreatedAt, &entry.Rank)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard rows: %w", err)
	}

	return entries, nil
}

// GetLeaderboardByMode retrieves leaderboard entries for a specific game mode
func (p *PostgresDB) GetLeaderboardByMode(leaderboardType models.LeaderboardType, gameMode models.GameMode, limit int) ([]models.LeaderboardEntry, error) {
	var query string
	var args []interface{}

	baseQuery := `
		SELECT
			g.user_id,
			u.name as user_name,
			u.avatar as user_avatar,
			g.score,
			g.id as game_id,
			g.created_at,
			ROW_NUMBER() OVER (ORDER BY g.score DESC) as rank
		FROM (
			SELECT
				user_id,
				MAX(score) as score,
				(ARRAY_AGG(id ORDER BY score DESC))[1] as id,
				(ARRAY_AGG(created_at ORDER BY score DESC))[1] as created_at
			FROM games
			WHERE (game_over = true OR victory = true) AND game_mode = $1`

	var timeFilter string
	switch leaderboardType {
	case models.LeaderboardDaily:
		timeFilter = ` AND created_at >= CURRENT_DATE`
	case models.LeaderboardWeekly:
		timeFilter = ` AND created_at >= DATE_TRUNC('week', CURRENT_DATE)`
	case models.LeaderboardMonthly:
		timeFilter = ` AND created_at >= DATE_TRUNC('month', CURRENT_DATE)`
	case models.LeaderboardAll:
		timeFilter = ""
	default:
		return nil, fmt.Errorf("invalid leaderboard type")
	}

	query = baseQuery + timeFilter + `
			GROUP BY user_id
		) g
		JOIN users u ON g.user_id = u.id
		ORDER BY g.score DESC LIMIT $2`
	args = append(args, string(gameMode), limit)

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard by mode: %w", err)
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.UserID, &entry.UserName, &entry.UserAvatar,
			&entry.Score, &entry.GameID, &entry.CreatedAt, &entry.Rank)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entry.GameMode = gameMode
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard rows: %w", err)
	}

	return entries, nil
}
