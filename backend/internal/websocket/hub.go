package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"game2048/internal/auth"
	"game2048/internal/cache"
	"game2048/internal/database"
	"game2048/internal/game"
	"game2048/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Game engine
	gameEngine *game.Engine

	// Database
	db database.Database

	// Cache for game sessions
	cache cache.Cache

	// Auth service
	authService *auth.AuthService

	// Mutex for thread safety
	mutex sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// User ID
	userID string

	// Current game ID
	gameID uuid.UUID

	// Hub reference
	hub *Hub
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, you should check the origin properly
		return true
	},
}

// NewHub creates a new WebSocket hub
func NewHub(gameEngine *game.Engine, db database.Database, authService *auth.AuthService, redisCache cache.Cache) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		gameEngine:  gameEngine,
		db:          db,
		cache:       redisCache,
		authService: authService,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client connected: %s", client.userID)

			// Send current game state if user has an active game
			go h.sendCurrentGameState(client)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client disconnected: %s", client.userID)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(c *gin.Context) {
	// Get JWT token from query parameter or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
		return
	}

	// Validate JWT token
	userID, err := h.authService.ValidateJWT(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create new client
	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		hub:    h,
	}

	// Register client
	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// sendCurrentGameState sends the current game state to a newly connected client
func (h *Hub) sendCurrentGameState(client *Client) {
	// Try to get game state from Redis cache first
	var gameState *models.GameState
	var err error

	if h.cache != nil {
		gameState, err = h.cache.GetGameSession(client.userID)
		if err != nil {
			// Cache miss or error, try database as fallback
			gameState, err = h.db.GetUserActiveGame(client.userID)
			if err != nil {
				log.Printf("Error getting active game for user %s: %v", client.userID, err)
				return
			}

			// If found in database, cache it for future use
			if gameState != nil && h.cache != nil {
				// Cache for 1 hour
				if err := h.cache.SetGameSession(client.userID, gameState, time.Hour); err != nil {
					log.Printf("Failed to cache game session: %v", err)
				}
			}
		}
	} else {
		// No cache available, use database
		gameState, err = h.db.GetUserActiveGame(client.userID)
		if err != nil {
			log.Printf("Error getting active game for user %s: %v", client.userID, err)
			return
		}
	}

	if gameState != nil {
		client.gameID = gameState.ID
		response := models.GameResponse{
			Board:    gameState.Board,
			Score:    gameState.Score,
			GameOver: gameState.GameOver,
			Victory:  gameState.Victory,
		}

		message := models.WebSocketMessage{
			Type: "game_state",
			Data: response,
		}

		client.sendMessage(message)
	}
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(message models.WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		close(c.send)
		c.hub.mutex.Lock()
		delete(c.hub.clients, c)
		c.hub.mutex.Unlock()
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message
		var message models.WebSocketMessage
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			log.Printf("Error parsing message: %v", err)
			c.sendError("Invalid message format")
			continue
		}

		// Handle message
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(message models.WebSocketMessage) {
	switch message.Type {
	case "move":
		c.handleMove(message.Data)
	case "new_game":
		c.handleNewGame(message.Data)
	case "get_leaderboard":
		c.handleGetLeaderboard(message.Data)
	default:
		c.sendError("Unknown message type")
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(errorMessage string) {
	response := models.ErrorResponse{
		Message: errorMessage,
	}

	message := models.WebSocketMessage{
		Type: "error",
		Data: response,
	}

	c.sendMessage(message)
}
