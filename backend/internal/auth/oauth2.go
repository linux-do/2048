package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"game2048/internal/cache"
	"game2048/internal/config"
	"game2048/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// OAuth2Provider represents an OAuth2 provider
type OAuth2Provider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
}

// UserInfo represents user information from OAuth2 provider
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar_url"`
	Provider string `json:"provider"`
}

// AuthService handles authentication
type AuthService struct {
	config   *config.Config
	provider OAuth2Provider
	cache    cache.Cache          // Redis cache for state management
	states   map[string]time.Time // Fallback for when Redis is not available
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg *config.Config, redisCache cache.Cache) (*AuthService, error) {
	var provider OAuth2Provider
	var err error

	// Only support custom provider
	provider, err = NewCustomProvider(cfg)

	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 provider: %w", err)
	}

	return &AuthService{
		config:   cfg,
		provider: provider,
		cache:    redisCache,
		states:   make(map[string]time.Time), // Fallback when Redis is not available
	}, nil
}

// GetAuthURL generates an OAuth2 authorization URL
func (a *AuthService) GetAuthURL() (string, error) {
	state, err := a.generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state with expiration (5 minutes)
	if a.cache != nil {
		// Use Redis cache
		if err := a.cache.SetOAuth2State(state, 5*time.Minute); err != nil {
			// Fallback to in-memory storage
			a.states[state] = time.Now().Add(5 * time.Minute)
		}
	} else {
		// Fallback to in-memory storage
		a.states[state] = time.Now().Add(5 * time.Minute)
	}

	return a.provider.GetAuthURL(state), nil
}

// HandleCallback handles the OAuth2 callback
func (a *AuthService) HandleCallback(ctx context.Context, code, state string) (*models.User, string, error) {
	// Validate state
	if !a.validateState(state) {
		return nil, "", fmt.Errorf("invalid state parameter")
	}

	// Exchange code for token
	token, err := a.provider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := a.provider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user info: %w", err)
	}

	// Create user model
	user := &models.User{
		ID:         uuid.New().String(),
		Email:      userInfo.Email,
		Name:       userInfo.Name,
		Avatar:     userInfo.Avatar,
		Provider:   userInfo.Provider,
		ProviderID: userInfo.ID,
	}

	// Generate JWT token
	jwtToken, err := a.GenerateJWT(user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return user, jwtToken, nil
}

// GenerateJWT generates a JWT token for the user
func (a *AuthService) GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.config.Server.JWTSecret))
}

// ValidateJWT validates a JWT token and returns the user ID
func (a *AuthService) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.config.Server.JWTSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
		return "", fmt.Errorf("user_id not found in token")
	}

	return "", fmt.Errorf("invalid token")
}

// generateState generates a random state string
func (a *AuthService) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// validateState validates the state parameter
func (a *AuthService) validateState(state string) bool {
	// Try Redis cache first
	if a.cache != nil {
		if a.cache.ValidateOAuth2State(state) {
			return true
		}
	}

	// Fallback to in-memory storage
	expiration, exists := a.states[state]
	if !exists {
		return false
	}

	// Check if state has expired
	if time.Now().After(expiration) {
		delete(a.states, state)
		return false
	}

	// Remove used state
	delete(a.states, state)
	return true
}

// CustomProvider implements OAuth2Provider for custom OAuth2 services
type CustomProvider struct {
	config *oauth2.Config
	cfg    *config.Config
}

// NewCustomProvider creates a new custom OAuth2 provider
func NewCustomProvider(cfg *config.Config) (*CustomProvider, error) {
	if cfg.OAuth2.ClientID == "" || cfg.OAuth2.ClientSecret == "" {
		return nil, fmt.Errorf("OAuth2 client ID and secret must be configured")
	}

	if cfg.OAuth2.AuthURL == "" || cfg.OAuth2.TokenURL == "" {
		return nil, fmt.Errorf("OAuth2 auth URL and token URL must be configured")
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.OAuth2.ClientID,
		ClientSecret: cfg.OAuth2.ClientSecret,
		RedirectURL:  cfg.OAuth2.RedirectURL,
		Scopes:       cfg.OAuth2.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.OAuth2.AuthURL,
			TokenURL: cfg.OAuth2.TokenURL,
		},
	}

	return &CustomProvider{
		config: oauth2Config,
		cfg:    cfg,
	}, nil
}

// GetAuthURL returns the custom OAuth2 authorization URL
func (c *CustomProvider) GetAuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges the authorization code for a token
func (c *CustomProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.config.Exchange(ctx, code)
}

// GetUserInfo gets user information from custom OAuth2 provider
func (c *CustomProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	if c.cfg.OAuth2.UserInfoURL == "" {
		return nil, fmt.Errorf("user info URL not configured")
	}

	client := c.config.Client(ctx, token)
	resp, err := client.Get(c.cfg.OAuth2.UserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", resp.Status)
	}

	var userResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userResponse); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Extract user information based on field mappings
	userInfo := &UserInfo{
		Provider: c.cfg.OAuth2.Provider,
	}

	// Extract user ID
	if id, ok := c.extractField(userResponse, c.cfg.OAuth2.UserIDField); ok {
		userInfo.ID = fmt.Sprintf("%v", id)
	} else {
		return nil, fmt.Errorf("user ID field '%s' not found in response", c.cfg.OAuth2.UserIDField)
	}

	// Extract email
	if email, ok := c.extractField(userResponse, c.cfg.OAuth2.UserEmailField); ok {
		userInfo.Email = fmt.Sprintf("%v", email)
	}

	// Extract name
	if name, ok := c.extractField(userResponse, c.cfg.OAuth2.UserNameField); ok {
		userInfo.Name = fmt.Sprintf("%v", name)
	} else {
		// Fallback to email or ID if name is not available
		if userInfo.Email != "" {
			userInfo.Name = userInfo.Email
		} else {
			userInfo.Name = userInfo.ID
		}
	}

	// Extract avatar
	if avatar, ok := c.extractField(userResponse, c.cfg.OAuth2.UserAvatarField); ok {
		userInfo.Avatar = fmt.Sprintf("%v", avatar)
	}

	return userInfo, nil
}

// extractField extracts a field from the user response, supporting nested fields with dot notation
func (c *CustomProvider) extractField(data map[string]interface{}, fieldPath string) (interface{}, bool) {
	if fieldPath == "" {
		return nil, false
	}

	// Support nested field access with dot notation (e.g., "user.profile.name")
	fields := strings.Split(fieldPath, ".")
	current := data

	for i, field := range fields {
		if i == len(fields)-1 {
			// Last field, return the value
			if value, exists := current[field]; exists {
				return value, true
			}
			return nil, false
		} else {
			// Intermediate field, navigate deeper
			if next, exists := current[field]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return nil, false
				}
			} else {
				return nil, false
			}
		}
	}

	return nil, false
}
