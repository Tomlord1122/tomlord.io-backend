package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader with CORS settings
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from our frontend
		origin := r.Header.Get("Origin")

		// Get environment to determine allowed origins
		appEnv := os.Getenv("APP_ENV")

		// Development and Minikube environments
		if appEnv == "local" || appEnv == "minikube" {
			allowedOrigins := []string{
				"http://localhost:5173",
				"http://localhost:3000",
				"http://localhost:4173", // Vite preview
				"http://minikube.local",
				"http://192.168.49.2", // Minikube default IP
				"http://192.168.49.1", // Minikube alternative IP
			}

			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}

			// Allow any localhost origin for development
			if strings.HasPrefix(origin, "http://localhost:") {
				return true
			}

			// Allow any minikube IP origin
			if strings.HasPrefix(origin, "http://192.168.49.") {
				return true
			}
		}

		// Production environment
		if appEnv == "production" {
			// Get allowed origins from environment variables
			allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
			frontendURL := os.Getenv("FRONTEND_URL")

			var allowedOrigins []string

			if allowedOriginsEnv != "" {
				// Use ALLOWED_ORIGINS if set
				allowedOrigins = strings.Split(allowedOriginsEnv, ",")
			} else if frontendURL != "" {
				// Fallback to FRONTEND_URL if ALLOWED_ORIGINS is not set
				allowedOrigins = []string{frontendURL}
			} else {
				// Default fallback for production
				allowedOrigins = []string{
					"https://tomlord.fyi",
				}
			}

			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
		}

		// Log rejected origins for debugging
		log.Printf("WebSocket connection rejected from origin: %s (APP_ENV: %s)", origin, appEnv)
		return false
	},
	// Add buffer sizes for better performance
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Message types for WebSocket communication
type MessageType string

const (
	MessageTypeNewComment    MessageType = "new_comment"
	MessageTypeThumbUpdate   MessageType = "thumb_update"
	MessageTypeCommentUpdate MessageType = "comment_update"
	MessageTypeCommentDelete MessageType = "comment_delete"
	MessageTypePing          MessageType = "ping"
	MessageTypePong          MessageType = "pong"
)

// WebSocket message structure
type WSMessage struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
	Room    string      `json:"room,omitempty"` // post_slug or blog_slug
}

// Client connection
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub
	rooms    map[string]bool // which rooms this client is subscribed to
	userID   string          // for authentication
	lastPong time.Time       // for heartbeat tracking
	mutex    sync.RWMutex    // for thread-safe access to client state
}

// Hub manages client connections and rooms
type Hub struct {
	// Map of room -> clients in that room
	rooms map[string]map[*Client]bool

	// Register/unregister clients
	register   chan *Client
	unregister chan *Client

	// Broadcast message to a specific room
	broadcast chan WSMessage

	// Mutex for thread safety
	mutex sync.RWMutex
}

// Constants for WebSocket timeouts and intervals
const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Create new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSMessage),
	}
}

// Run the hub - handles client registration and message broadcasting
func (h *Hub) Run() {
	// Start cleanup routine for stale connections
	go h.cleanupStaleConnections()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			client.mutex.RLock()
			for room := range client.rooms {
				if h.rooms[room] == nil {
					h.rooms[room] = make(map[*Client]bool)
				}
				h.rooms[room][client] = true
			}
			client.mutex.RUnlock()
			h.mutex.Unlock()
			log.Printf("Client %s connected to rooms: %v", client.userID, client.rooms)

		case client := <-h.unregister:
			h.mutex.Lock()
			client.mutex.RLock()
			for room := range client.rooms {
				if clients, ok := h.rooms[room]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			client.mutex.RUnlock()

			// Close client's send channel if it's still open
			select {
			case <-client.send:
			default:
				close(client.send)
			}

			h.mutex.Unlock()
			log.Printf("Client %s disconnected from rooms: %v", client.userID, client.rooms)

		case message := <-h.broadcast:
			h.mutex.RLock()
			if clients, ok := h.rooms[message.Room]; ok {
				messageBytes, err := json.Marshal(message)
				if err != nil {
					log.Printf("Error marshaling message: %v", err)
					h.mutex.RUnlock()
					continue
				}

				for client := range clients {
					select {
					case client.send <- messageBytes:
					default:
						// Client's send channel is blocked, remove it
						delete(clients, client)
						close(client.send)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Cleanup stale connections periodically
func (h *Hub) cleanupStaleConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mutex.RLock()
		var staleClients []*Client

		for _, clients := range h.rooms {
			for client := range clients {
				client.mutex.RLock()
				if time.Since(client.lastPong) > pongWait {
					staleClients = append(staleClients, client)
				}
				client.mutex.RUnlock()
			}
		}
		h.mutex.RUnlock()

		// Remove stale clients
		for _, client := range staleClients {
			log.Printf("Removing stale client: %s", client.userID)
			h.unregister <- client
		}
	}
}

// Broadcast message to a specific room
func (h *Hub) BroadcastToRoom(room string, msgType MessageType, payload interface{}) {
	message := WSMessage{
		Type:    msgType,
		Payload: payload,
		Room:    room,
	}

	select {
	case h.broadcast <- message:
		log.Printf("Broadcasting %s message to room '%s'", msgType, room)
	default:
		log.Printf("Failed to broadcast message to room %s - channel full", room)
	}
}

// Handle WebSocket connection
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get rooms to subscribe to from query parameters
	rooms := make(map[string]bool)
	if roomsParam := r.URL.Query().Get("rooms"); roomsParam != "" {
		var roomsList []string
		if err := json.Unmarshal([]byte(roomsParam), &roomsList); err == nil {
			for _, room := range roomsList {
				rooms[room] = true
			}
		}
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		hub:      h,
		rooms:    rooms,
		userID:   userID,
		lastPong: time.Now(),
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Handle writing messages to WebSocket with heartbeat
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing message to client %s: %v", c.userID, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping to client %s: %v", c.userID, err)
				return
			}
		}
	}
}

// Handle reading messages from WebSocket with heartbeat handling
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.mutex.Lock()
		c.lastPong = time.Now()
		c.mutex.Unlock()
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg struct {
			Action string   `json:"action"`
			Rooms  []string `json:"rooms"`
		}

		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.userID, err)
			}
			break
		}

		// Handle room subscription changes
		switch msg.Action {
		case "subscribe":
			c.hub.mutex.Lock()
			c.mutex.Lock()
			for _, room := range msg.Rooms {
				c.rooms[room] = true
				if c.hub.rooms[room] == nil {
					c.hub.rooms[room] = make(map[*Client]bool)
				}
				c.hub.rooms[room][c] = true
			}
			c.mutex.Unlock()
			c.hub.mutex.Unlock()
			log.Printf("Client %s subscribed to rooms: %v", c.userID, msg.Rooms)
		case "unsubscribe":
			c.hub.mutex.Lock()
			c.mutex.Lock()
			for _, room := range msg.Rooms {
				delete(c.rooms, room)
				if clients, ok := c.hub.rooms[room]; ok {
					delete(clients, c)
				}
			}
			c.mutex.Unlock()
			c.hub.mutex.Unlock()
			log.Printf("Client %s unsubscribed from rooms: %v", c.userID, msg.Rooms)
		}
	}
}
