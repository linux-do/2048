package handlers

import (
	"net/http"

	"game2048/internal/auth"
	"game2048/internal/database"
	"game2048/internal/i18n"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService *auth.AuthService
	db          database.Database
	i18n        *i18n.I18n
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.AuthService, db database.Database, i18nManager *i18n.I18n) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		db:          db,
		i18n:        i18nManager,
	}
}

// Login initiates the OAuth2 login flow
func (h *AuthHandler) Login(c *gin.Context) {
	authURL, err := h.authService.GetAuthURL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate auth URL",
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// Callback handles the OAuth2 callback
func (h *AuthHandler) Callback(c *gin.Context) {
	lang := i18n.GetLanguage(c)
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	// Check for OAuth2 errors
	if errorParam != "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": h.i18n.Tf(lang, "error.something_wrong") + ": " + errorParam,
			"lang":  lang,
		})
		return
	}

	if code == "" || state == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": h.i18n.T(lang, "error.something_wrong"),
			"lang":  lang,
		})
		return
	}

	// Handle the callback
	user, token, err := h.authService.HandleCallback(c.Request.Context(), code, state)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": h.i18n.T(lang, "error.something_wrong"),
			"lang":  lang,
		})
		return
	}

	// Check if user exists in database
	existingUser, err := h.db.GetUserByProvider(user.Provider, user.ProviderID)
	if err != nil {
		// User doesn't exist, create new user
		if err := h.db.CreateUser(user); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": h.i18n.T(lang, "error.something_wrong"),
				"lang":  lang,
			})
			return
		}
	} else {
		// User exists, update user info but keep the existing ID
		user.ID = existingUser.ID
		user.CreatedAt = existingUser.CreatedAt
		if err := h.db.CreateUser(user); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": h.i18n.T(lang, "error.something_wrong"),
				"lang":  lang,
			})
			return
		}
	}

	// Generate JWT token with the correct user ID (either new or existing)
	token, err = h.authService.GenerateJWT(user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": h.i18n.T(lang, "error.something_wrong"),
			"lang":  lang,
		})
		return
	}

	// Set JWT token as HTTP-only cookie
	c.SetCookie(
		"auth_token",
		token,
		3600*24, // 24 hours
		"/",
		"",
		h.isHTTPS(c), // Secure flag based on HTTPS detection
		true,         // HTTP-only
	)

	// Redirect to game page
	c.HTML(http.StatusOK, "login_success.html", gin.H{
		"user":  user,
		"token": token,
		"lang":  i18n.GetLanguage(c),
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Clear the auth cookie
	c.SetCookie(
		"auth_token",
		"",
		-1,
		"/",
		"",
		h.isHTTPS(c), // Same secure flag as when setting
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// Me returns the current user information
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	user, err := h.db.GetUser(userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// AuthMiddleware validates JWT tokens
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from cookie first
		token, err := c.Cookie("auth_token")
		if err != nil {
			// Try to get token from Authorization header
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Missing authentication token",
				})
				c.Abort()
				return
			}
		}

		// Validate token
		userID, err := h.authService.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authentication token",
			})
			c.Abort()
			return
		}

		// Set user ID in context
		c.Set("user_id", userID)
		c.Next()
	}
}

// OptionalAuthMiddleware validates JWT tokens but doesn't require them
func (h *AuthHandler) OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from cookie first
		token, err := c.Cookie("auth_token")
		if err != nil {
			// Try to get token from Authorization header
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			} else {
				// No token found, continue without authentication
				c.Next()
				return
			}
		}

		// Validate token
		userID, err := h.authService.ValidateJWT(token)
		if err != nil {
			// Invalid token, continue without authentication
			c.Next()
			return
		}

		// Set user ID in context
		c.Set("user_id", userID)
		c.Next()
	}
}

// isHTTPS determines if the request is using HTTPS
// Checks TLS connection, X-Forwarded-Proto header, and X-Forwarded-Ssl header
func (h *AuthHandler) isHTTPS(c *gin.Context) bool {
	return c.Request.TLS != nil ||
		c.GetHeader("X-Forwarded-Proto") == "https" ||
		c.GetHeader("X-Forwarded-Ssl") == "on"
}
