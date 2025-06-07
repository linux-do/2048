package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"game2048/internal/auth"
	"game2048/internal/cache"
	"game2048/internal/config"
	"game2048/internal/database"
	"game2048/internal/game"
	"game2048/internal/handlers"
	"game2048/internal/i18n"
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

	// Initialize i18n
	i18nManager := i18n.New(cfg.I18n.DefaultLanguage)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, db, i18nManager)
	leaderboardHandler := handlers.NewLeaderboardHandler(db, redisCache)

	// Create Gin router
	router := gin.Default()

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.Server.CORSOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Use i18n middleware
	router.Use(i18n.Middleware(i18nManager))

	// Create template functions
	createTemplateFuncs := func(lang string) template.FuncMap {
		funcMap := template.FuncMap{
			"static": func(path string) string {
				// In development mode, refresh version on each request
				if !cfg.Server.StaticFilesEmbedded {
					versionManager.RefreshVersion(path)
				}
				return versionManager.GetVersionedURL("/static" + path)
			},
		}
		
		// Add i18n functions
		i18nFuncs := i18nManager.TemplateFuncMap(lang)
		for k, v := range i18nFuncs {
			funcMap[k] = v
		}
		
		return funcMap
	}

	// Load HTML templates
	if cfg.Server.StaticFilesEmbedded {
		// Load embedded templates with custom functions
		tmpl := template.Must(template.New("").Funcs(createTemplateFuncs("en")).ParseFS(templateFiles, "templates/*.html"))
		router.SetHTMLTemplate(tmpl)

		// Serve embedded static files - need to use sub filesystem to strip the "static" prefix
		staticFS, err := fs.Sub(staticFiles, "static")
		if err != nil {
			log.Fatalf("Failed to create static sub filesystem: %v", err)
		}
		router.StaticFS("/static", http.FS(staticFS))
	} else {
		// Load templates from file system (development mode) with custom functions
		tmpl := template.Must(template.New("").Funcs(createTemplateFuncs("en")).ParseGlob("cmd/server/templates/*.html"))
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

	// Language switching route
	router.GET("/lang/:lang", i18n.SetLanguage(i18nManager))

	// API endpoint for getting supported languages
	router.GET("/api/languages", func(c *gin.Context) {
		languages := make([]gin.H, 0)
		for _, lang := range i18nManager.GetSupportedLanguages() {
			languages = append(languages, gin.H{
				"code": lang,
				"name": i18nManager.GetLanguageName(lang),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"languages": languages,
		})
	})

	// API endpoint for getting translations for client-side use
	router.GET("/api/translations/:lang", func(c *gin.Context) {
		lang := c.Param("lang")
		
		// Validate language
		supported := false
		for _, supportedLang := range i18nManager.GetSupportedLanguages() {
			if supportedLang == lang {
				supported = true
				break
			}
		}
		
		if !supported {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported language"})
			return
		}
		
		// Get client-side translations (only keys needed by JavaScript)
		clientKeys := []string{
			"game.victory_message",
			"game.game_over_message",
			"game.connecting",
			"game.connected", 
			"game.disconnected",
			"websocket.not_authenticated",
			"websocket.connection_failed",
			"websocket.connection_lost",
			"websocket.not_connected",
			"websocket.connection_error",
			"errors.initialization_failed",
			"errors.game_load_failed",
			"errors.refresh_page",
			"errors.unexpected_error",
			"errors.network_error",
			"leaderboard.loading",
			"leaderboard.no_scores",
			"leaderboard.be_first",
			"leaderboard.failed_to_load",
			"common.loading",
		}
		
		translations := make(map[string]string)
		for _, key := range clientKeys {
			translations[key] = i18nManager.T(lang, key)
		}
		
		c.JSON(http.StatusOK, gin.H{
			"language": lang,
			"translations": translations,
		})
	})

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
		lang := i18n.GetLanguage(c)
		c.HTML(http.StatusOK, "leaderboard.html", gin.H{
			"title": i18nManager.T(lang, "leaderboard.title"),
			"lang":  lang,
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
		lang := i18n.GetLanguage(c)
		userID, exists := c.Get("user_id")
		if !exists {
			// User not authenticated, show login page
			c.HTML(http.StatusOK, "login.html", gin.H{
				"title": i18nManager.T(lang, "game.title"),
				"lang":  lang,
			})
			return
		}

		// User authenticated, show game page
		user, err := db.GetUser(userID.(string))
		if err != nil {
			log.Printf("Failed to get user: %v", err)
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": i18nManager.T(lang, "error.something_wrong"),
				"lang":  lang,
			})
			return
		}

		c.HTML(http.StatusOK, "game.html", gin.H{
			"title": i18nManager.T(lang, "game.title"),
			"user":  user,
			"lang":  lang,
		})
	})

	// Start server
	log.Printf("Starting server on %s", cfg.GetServerAddress())
	if err := router.Run(cfg.GetServerAddress()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
