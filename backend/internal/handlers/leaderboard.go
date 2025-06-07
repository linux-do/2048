package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"game2048/internal/cache"
	"game2048/internal/database"
	"game2048/pkg/models"

	"github.com/gin-gonic/gin"
)

// LeaderboardHandler handles leaderboard-related requests
type LeaderboardHandler struct {
	db    database.Database
	cache cache.Cache
}

// NewLeaderboardHandler creates a new leaderboard handler
func NewLeaderboardHandler(db database.Database, redisCache cache.Cache) *LeaderboardHandler {
	return &LeaderboardHandler{
		db:    db,
		cache: redisCache,
	}
}

// GetLeaderboard handles public leaderboard requests
func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	// Get leaderboard type from query parameter
	leaderboardType := c.DefaultQuery("type", "daily")

	// Validate leaderboard type
	var lbType models.LeaderboardType
	switch leaderboardType {
	case "daily":
		lbType = models.LeaderboardDaily
	case "weekly":
		lbType = models.LeaderboardWeekly
	case "monthly":
		lbType = models.LeaderboardMonthly
	case "all":
		lbType = models.LeaderboardAll
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid leaderboard type. Must be one of: daily, weekly, monthly, all",
		})
		return
	}

	// Get game mode from query parameter
	gameModeStr := c.DefaultQuery("game_mode", "classic")
	var gameMode models.GameMode
	switch gameModeStr {
	case "classic":
		gameMode = models.GameModeClassic
	case "challenge":
		gameMode = models.GameModeChallenge
	default:
		gameMode = models.GameModeClassic
	}

	// Get limit from query parameter (default 100, max 100)
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 100
	}

	// Try to get from cache first (for now, skip cache for game mode specific queries)
	var entries []models.LeaderboardEntry

	// For now, always get from database to support game mode filtering
	// TODO: Update cache to support game mode keys
	entries, err = h.db.GetLeaderboardByMode(lbType, gameMode, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get leaderboard",
		})
		return
	}

	// TODO: Cache the result if cache is available (update cache to support game modes)
	// For now, skip caching for game mode specific queries

	// Return response
	response := models.LeaderboardResponse{
		Type:     lbType,
		Rankings: entries,
	}

	c.JSON(http.StatusOK, response)
}

// RefreshCache manually refreshes the leaderboard cache
// Only accessible by user with ID "1" (admin)
func (h *LeaderboardHandler) RefreshCache(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	// Check if user is admin (ID = "1")
	if userID.(string) != "1" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied. Admin privileges required.",
		})
		return
	}

	// Check if cache is available
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache service not available",
		})
		return
	}

	// Get the type parameter (optional)
	typeParam := c.Query("type")

	var refreshedTypes []string
	var errors []string

	if typeParam != "" {
		// Refresh specific leaderboard type
		lbType := models.LeaderboardType(typeParam)

		// Validate leaderboard type
		if lbType != models.LeaderboardDaily && lbType != models.LeaderboardWeekly && lbType != models.LeaderboardMonthly && lbType != models.LeaderboardAll {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid leaderboard type. Must be 'daily', 'weekly', 'monthly', or 'all'",
			})
			return
		}

		// Invalidate cache for this type
		if err := h.cache.InvalidateLeaderboard(lbType); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to invalidate %s cache: %v", lbType, err))
		} else {
			refreshedTypes = append(refreshedTypes, string(lbType))
		}
	} else {
		// Refresh all leaderboard types
		allTypes := []models.LeaderboardType{
			models.LeaderboardDaily,
			models.LeaderboardWeekly,
			models.LeaderboardMonthly,
			models.LeaderboardAll,
		}

		for _, lbType := range allTypes {
			if err := h.cache.InvalidateLeaderboard(lbType); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to invalidate %s cache: %v", lbType, err))
			} else {
				refreshedTypes = append(refreshedTypes, string(lbType))
			}
		}
	}

	// Prepare response
	response := gin.H{
		"message":         "Cache refresh completed",
		"refreshed_types": refreshedTypes,
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["message"] = "Cache refresh completed with some errors"
	}

	// Return appropriate status code
	if len(refreshedTypes) > 0 {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusInternalServerError, response)
	}
}
