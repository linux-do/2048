package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system using GORM
type GormUser struct {
	ID         string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Email      string    `gorm:"type:varchar(255);not null" json:"email"`
	Name       string    `gorm:"type:varchar(255);not null" json:"name"`
	Avatar     string    `gorm:"type:varchar(500)" json:"avatar"`
	Provider   string    `gorm:"type:varchar(50);not null" json:"provider"`
	ProviderID string    `gorm:"type:varchar(255);not null" json:"provider_id"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Games []GormGame `gorm:"foreignKey:UserID" json:"games,omitempty"`
}

// TableName specifies the table name for GormUser
func (GormUser) TableName() string {
	return "users"
}

// ToUser converts GormUser to User
func (gu *GormUser) ToUser() *User {
	return &User{
		ID:         gu.ID,
		Email:      gu.Email,
		Name:       gu.Name,
		Avatar:     gu.Avatar,
		Provider:   gu.Provider,
		ProviderID: gu.ProviderID,
		CreatedAt:  gu.CreatedAt,
		UpdatedAt:  gu.UpdatedAt,
	}
}

// FromUser converts User to GormUser
func (gu *GormUser) FromUser(u *User) {
	gu.ID = u.ID
	gu.Email = u.Email
	gu.Name = u.Name
	gu.Avatar = u.Avatar
	gu.Provider = u.Provider
	gu.ProviderID = u.ProviderID
	gu.CreatedAt = u.CreatedAt
	gu.UpdatedAt = u.UpdatedAt
}

// GormGame represents a game session using GORM
type GormGame struct {
	ID           uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       string            `gorm:"type:varchar(255);not null;index" json:"user_id"`
	Board        BoardJSON         `gorm:"type:jsonb;not null" json:"board"`
	Score        int               `gorm:"not null;default:0;index:idx_games_score" json:"score"`
	GameOver     bool              `gorm:"not null;default:false" json:"game_over"`
	Victory      bool              `gorm:"not null;default:false" json:"victory"`
	GameMode     string            `gorm:"type:varchar(20);not null;default:'classic';index:idx_games_mode" json:"game_mode"`
	DisabledCell *DisabledCellJSON `gorm:"type:jsonb" json:"disabled_cell"`
	CreatedAt    time.Time         `gorm:"autoCreateTime;index:idx_games_created_at" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	User GormUser `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName specifies the table name for GormGame
func (GormGame) TableName() string {
	return "games"
}

// BoardJSON is a custom type for handling JSON serialization of the game board
type BoardJSON Board

// Scan implements the sql.Scanner interface for reading from database
func (b *BoardJSON) Scan(value interface{}) error {
	if value == nil {
		*b = BoardJSON{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into BoardJSON", value)
	}

	var board Board
	if err := json.Unmarshal(bytes, &board); err != nil {
		return err
	}

	*b = BoardJSON(board)
	return nil
}

// Value implements the driver.Valuer interface for writing to database
func (b BoardJSON) Value() (driver.Value, error) {
	return json.Marshal(Board(b))
}

// DisabledCellJSON is a custom type for handling JSON serialization of disabled cells
type DisabledCellJSON DisabledCell

// Scan implements the sql.Scanner interface for reading from database
func (dc *DisabledCellJSON) Scan(value interface{}) error {
	if value == nil {
		*dc = DisabledCellJSON{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into DisabledCellJSON", value)
	}

	var disabledCell DisabledCell
	if err := json.Unmarshal(bytes, &disabledCell); err != nil {
		return err
	}

	*dc = DisabledCellJSON(disabledCell)
	return nil
}

// Value implements the driver.Valuer interface for writing to database
func (dc DisabledCellJSON) Value() (driver.Value, error) {
	return json.Marshal(DisabledCell(dc))
}

// ToGameState converts GormGame to GameState
func (gg *GormGame) ToGameState() *GameState {
	var disabledCell *DisabledCell
	if gg.DisabledCell != nil {
		dc := DisabledCell(*gg.DisabledCell)
		disabledCell = &dc
	}

	return &GameState{
		ID:           gg.ID,
		UserID:       gg.UserID,
		Board:        Board(gg.Board),
		Score:        gg.Score,
		GameOver:     gg.GameOver,
		Victory:      gg.Victory,
		GameMode:     GameMode(gg.GameMode),
		DisabledCell: disabledCell,
		CreatedAt:    gg.CreatedAt,
		UpdatedAt:    gg.UpdatedAt,
	}
}

// FromGameState converts GameState to GormGame
func (gg *GormGame) FromGameState(gs *GameState) {
	gg.ID = gs.ID
	gg.UserID = gs.UserID
	gg.Board = BoardJSON(gs.Board)
	gg.Score = gs.Score
	gg.GameOver = gs.GameOver
	gg.Victory = gs.Victory
	gg.GameMode = string(gs.GameMode)

	if gs.DisabledCell != nil {
		dc := DisabledCellJSON(*gs.DisabledCell)
		gg.DisabledCell = &dc
	} else {
		gg.DisabledCell = nil
	}

	gg.CreatedAt = gs.CreatedAt
	gg.UpdatedAt = gs.UpdatedAt
}

// GormLeaderboardEntry represents a leaderboard entry using GORM
type GormLeaderboardEntry struct {
	UserID     string    `gorm:"type:varchar(255);not null" json:"user_id"`
	UserName   string    `gorm:"type:varchar(255);not null" json:"user_name"`
	UserAvatar string    `gorm:"type:varchar(500)" json:"user_avatar"`
	Score      int       `gorm:"not null" json:"score"`
	Rank       int       `gorm:"not null" json:"rank"`
	GameID     uuid.UUID `gorm:"type:uuid;not null" json:"game_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// ToLeaderboardEntry converts GormLeaderboardEntry to LeaderboardEntry
func (gle *GormLeaderboardEntry) ToLeaderboardEntry() *LeaderboardEntry {
	return &LeaderboardEntry{
		UserID:     gle.UserID,
		UserName:   gle.UserName,
		UserAvatar: gle.UserAvatar,
		Score:      gle.Score,
		Rank:       gle.Rank,
		GameID:     gle.GameID,
		CreatedAt:  gle.CreatedAt,
	}
}

// Daily leaderboard cache table
type GormDailyLeaderboard struct {
	UserID     string    `gorm:"type:varchar(255);not null;primaryKey" json:"user_id"`
	UserName   string    `gorm:"type:varchar(255);not null" json:"user_name"`
	UserAvatar string    `gorm:"type:varchar(500)" json:"user_avatar"`
	Score      int       `gorm:"not null;index:idx_daily_score" json:"score"`
	Rank       int       `gorm:"not null" json:"rank"`
	GameID     uuid.UUID `gorm:"type:uuid;not null" json:"game_id"`
	GameMode   string    `gorm:"type:varchar(20);not null;default:'classic';primaryKey" json:"game_mode"`
	Date       time.Time `gorm:"type:date;not null;primaryKey;default:CURRENT_DATE" json:"date"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (GormDailyLeaderboard) TableName() string {
	return "leaderboard_daily"
}

// Weekly leaderboard cache table
type GormWeeklyLeaderboard struct {
	UserID     string    `gorm:"type:varchar(255);not null;primaryKey" json:"user_id"`
	UserName   string    `gorm:"type:varchar(255);not null" json:"user_name"`
	UserAvatar string    `gorm:"type:varchar(500)" json:"user_avatar"`
	Score      int       `gorm:"not null;index:idx_weekly_score" json:"score"`
	Rank       int       `gorm:"not null" json:"rank"`
	GameID     uuid.UUID `gorm:"type:uuid;not null" json:"game_id"`
	GameMode   string    `gorm:"type:varchar(20);not null;default:'classic';primaryKey" json:"game_mode"`
	WeekStart  time.Time `gorm:"type:date;not null;primaryKey" json:"week_start"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (GormWeeklyLeaderboard) TableName() string {
	return "leaderboard_weekly"
}

// Monthly leaderboard cache table
type GormMonthlyLeaderboard struct {
	UserID     string    `gorm:"type:varchar(255);not null;primaryKey" json:"user_id"`
	UserName   string    `gorm:"type:varchar(255);not null" json:"user_name"`
	UserAvatar string    `gorm:"type:varchar(500)" json:"user_avatar"`
	Score      int       `gorm:"not null;index:idx_monthly_score" json:"score"`
	Rank       int       `gorm:"not null" json:"rank"`
	GameID     uuid.UUID `gorm:"type:uuid;not null" json:"game_id"`
	GameMode   string    `gorm:"type:varchar(20);not null;default:'classic';primaryKey" json:"game_mode"`
	MonthStart time.Time `gorm:"type:date;not null;primaryKey" json:"month_start"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (GormMonthlyLeaderboard) TableName() string {
	return "leaderboard_monthly"
}
