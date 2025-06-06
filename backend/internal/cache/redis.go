package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"game2048/internal/config"
	"game2048/pkg/models"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements caching using Redis
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// Cache interface defines caching operations
type Cache interface {
	// Session management
	SetSession(key string, value interface{}, expiration time.Duration) error
	GetSession(key string, dest interface{}) error
	DeleteSession(key string) error

	// OAuth2 state management
	SetOAuth2State(state string, expiration time.Duration) error
	ValidateOAuth2State(state string) bool

	// Leaderboard caching
	SetLeaderboard(leaderboardType models.LeaderboardType, entries []models.LeaderboardEntry, expiration time.Duration) error
	GetLeaderboard(leaderboardType models.LeaderboardType) ([]models.LeaderboardEntry, error)
	InvalidateLeaderboard(leaderboardType models.LeaderboardType) error

	// Game session caching
	SetGameSession(userID string, game *models.GameState, expiration time.Duration) error
	GetGameSession(userID string) (*models.GameState, error)
	DeleteGameSession(userID string) error

	// JWT blacklist
	BlacklistJWT(tokenID string, expiration time.Duration) error
	IsJWTBlacklisted(tokenID string) bool

	// Generic operations
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string, dest interface{}) error
	Delete(key string) error
	Exists(key string) bool
	Close() error
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg *config.Config) (*RedisCache, error) {
	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis")

	return &RedisCache{
		client: rdb,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Set stores a value in Redis
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(r.ctx, key, data, expiration).Err()
}

// Get retrieves a value from Redis
func (r *RedisCache) Get(key string, dest interface{}) error {
	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found")
		}
		return fmt.Errorf("failed to get value: %w", err)
	}

	return json.Unmarshal([]byte(data), dest)
}

// Delete removes a key from Redis
func (r *RedisCache) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (r *RedisCache) Exists(key string) bool {
	result, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

// SetSession stores a session value
func (r *RedisCache) SetSession(key string, value interface{}, expiration time.Duration) error {
	sessionKey := fmt.Sprintf("session:%s", key)
	return r.Set(sessionKey, value, expiration)
}

// GetSession retrieves a session value
func (r *RedisCache) GetSession(key string, dest interface{}) error {
	sessionKey := fmt.Sprintf("session:%s", key)
	return r.Get(sessionKey, dest)
}

// DeleteSession removes a session
func (r *RedisCache) DeleteSession(key string) error {
	sessionKey := fmt.Sprintf("session:%s", key)
	return r.Delete(sessionKey)
}

// SetOAuth2State stores an OAuth2 state
func (r *RedisCache) SetOAuth2State(state string, expiration time.Duration) error {
	stateKey := fmt.Sprintf("oauth2:state:%s", state)
	return r.client.Set(r.ctx, stateKey, "valid", expiration).Err()
}

// ValidateOAuth2State validates and removes an OAuth2 state
func (r *RedisCache) ValidateOAuth2State(state string) bool {
	stateKey := fmt.Sprintf("oauth2:state:%s", state)

	// Use a Lua script to atomically check and delete
	script := `
		if redis.call("exists", KEYS[1]) == 1 then
			redis.call("del", KEYS[1])
			return 1
		else
			return 0
		end
	`

	result, err := r.client.Eval(r.ctx, script, []string{stateKey}).Result()
	if err != nil {
		return false
	}

	return result.(int64) == 1
}

// SetLeaderboard caches leaderboard entries
func (r *RedisCache) SetLeaderboard(leaderboardType models.LeaderboardType, entries []models.LeaderboardEntry, expiration time.Duration) error {
	leaderboardKey := fmt.Sprintf("leaderboard:%s", string(leaderboardType))
	return r.Set(leaderboardKey, entries, expiration)
}

// GetLeaderboard retrieves cached leaderboard entries
func (r *RedisCache) GetLeaderboard(leaderboardType models.LeaderboardType) ([]models.LeaderboardEntry, error) {
	leaderboardKey := fmt.Sprintf("leaderboard:%s", string(leaderboardType))
	var entries []models.LeaderboardEntry
	err := r.Get(leaderboardKey, &entries)
	return entries, err
}

// InvalidateLeaderboard removes cached leaderboard
func (r *RedisCache) InvalidateLeaderboard(leaderboardType models.LeaderboardType) error {
	leaderboardKey := fmt.Sprintf("leaderboard:%s", string(leaderboardType))
	return r.Delete(leaderboardKey)
}

// SetGameSession caches a game session
func (r *RedisCache) SetGameSession(userID string, game *models.GameState, expiration time.Duration) error {
	gameKey := fmt.Sprintf("game:session:%s", userID)
	return r.Set(gameKey, game, expiration)
}

// GetGameSession retrieves a cached game session
func (r *RedisCache) GetGameSession(userID string) (*models.GameState, error) {
	gameKey := fmt.Sprintf("game:session:%s", userID)
	var game models.GameState
	err := r.Get(gameKey, &game)
	return &game, err
}

// DeleteGameSession removes a game session
func (r *RedisCache) DeleteGameSession(userID string) error {
	gameKey := fmt.Sprintf("game:session:%s", userID)
	return r.Delete(gameKey)
}

// BlacklistJWT adds a JWT token to the blacklist
func (r *RedisCache) BlacklistJWT(tokenID string, expiration time.Duration) error {
	blacklistKey := fmt.Sprintf("jwt:blacklist:%s", tokenID)
	return r.client.Set(r.ctx, blacklistKey, "blacklisted", expiration).Err()
}

// IsJWTBlacklisted checks if a JWT token is blacklisted
func (r *RedisCache) IsJWTBlacklisted(tokenID string) bool {
	blacklistKey := fmt.Sprintf("jwt:blacklist:%s", tokenID)
	return r.Exists(blacklistKey)
}
