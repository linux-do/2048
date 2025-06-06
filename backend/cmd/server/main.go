package main

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game2048/internal/auth"
	"game2048/internal/cache"
	"game2048/internal/config"
	"game2048/internal/database"
	"game2048/internal/game"
	"game2048/internal/handlers"
	"game2048/internal/version"
	"game2048/internal/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize database with GORM
	db, err := database.NewGormDB(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis cache (optional)
	var redisCache cache.Cache
	redisCache, err = cache.NewRedisCache(cfg)
	if err != nil {
		log.Printf("Failed to connect to Redis, continuing without cache: %v", err)
		redisCache = nil
	}
	if redisCache != nil {
		defer redisCache.Close()
		log.Println("Redis cache initialized successfully")
	}

	// Initialize auth service
	authService, err := auth.NewAuthService(cfg, redisCache)
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}

	// Initialize game engine
	gameEngine := game.NewEngine()

	// Initialize WebSocket hub
	hub := websocket.NewHub(gameEngine, db, authService, redisCache)
	go hub.Run()

	// Initialize version manager for static files
	versionManager := version.NewManager("cmd/server/static")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, db)
	leaderboardHandler := handlers.NewLeaderboardHandler(db, redisCache)

	// Create Gin router
	router := gin.Default()

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.Server.CORSOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Create template functions
	funcMap := template.FuncMap{
		"static": func(path string) string {
			// In development mode, refresh version on each request
			if !cfg.Server.StaticFilesEmbedded {
				versionManager.RefreshVersion(path)
			}
			return versionManager.GetVersionedURL("/static" + path)
		},
	}

	// Load HTML templates
	if cfg.Server.StaticFilesEmbedded {
		// Load embedded templates with custom functions
		tmpl := template.Must(template.New("").Funcs(funcMap).ParseFS(templateFiles, "templates/*.html"))
		router.SetHTMLTemplate(tmpl)

		// Serve embedded static files - need to use sub filesystem to strip the "static" prefix
		staticFS, err := fs.Sub(staticFiles, "static")
		if err != nil {
			log.Fatalf("Failed to create static sub filesystem: %v", err)
		}
		router.StaticFS("/static", http.FS(staticFS))
	} else {
		// Load templates from file system (development mode) with custom functions
		tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("cmd/server/templates/*.html"))
		router.SetHTMLTemplate(tmpl)
		router.Static("/static", "cmd/server/static")
	}

	// Health check endpoint
	if cfg.Server.EnableHealthCheck {
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "healthy",
				"service": "game2048",
			})
		})
	}

	// Authentication routes
	authRoutes := router.Group("/auth")
	{
		authRoutes.GET("/login", authHandler.Login)
		authRoutes.GET("/callback", authHandler.Callback)
		authRoutes.POST("/logout", authHandler.Logout)
		authRoutes.GET("/me", authHandler.AuthMiddleware(), authHandler.Me)
	}

	// Public pages
	router.GET("/leaderboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "leaderboard.html", gin.H{
			"title": "2048 Game - Leaderboards",
		})
	})

	// WebSocket endpoint
	router.GET("/ws", hub.HandleWebSocket)

	// Public API routes (no authentication required)
	publicAPI := router.Group("/api/public")
	{
		publicAPI.GET("/leaderboard", leaderboardHandler.GetLeaderboard)
	}

	// API routes (protected)
	apiRoutes := router.Group("/api")
	apiRoutes.Use(authHandler.AuthMiddleware())
	{
		// Admin endpoints
		apiRoutes.GET("/admin/refresh-cache", leaderboardHandler.RefreshCache)

		// Game endpoints could be added here if needed
		// For now, all game logic is handled via WebSocket
	}

	// Serve the main game page
	router.GET("/", authHandler.OptionalAuthMiddleware(), func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			// User not authenticated, show login page
			c.HTML(http.StatusOK, "login.html", gin.H{
				"title": "2048 Game - Login",
			})
			return
		}

		// User authenticated, show game page
		user, err := db.GetUser(userID.(string))
		if err != nil {
			log.Printf("Failed to get user: %v", err)
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": "Failed to load user data",
			})
			return
		}

		c.HTML(http.StatusOK, "game.html", gin.H{
			"title": "2048 Game",
			"user":  user,
		})
	})

	// Start server with graceful shutdown
	srv := &http.Server{
		Addr:    cfg.GetServerAddress(),
		Handler: router,
	}

	go func() {
		log.Printf("Starting server on %s", cfg.GetServerAddress())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
