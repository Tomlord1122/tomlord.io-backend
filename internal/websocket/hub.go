package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader with CORS settings
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from our frontend
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:5173" || origin == "http://localhost:3000"
	},
}

// Message types for WebSocket communication
type MessageType string

const (
	MessageTypeNewComment    MessageType = "new_comment"
	MessageTypeThumbUpdate   MessageType = "thumb_update"
	MessageTypeCommentUpdate MessageType = "comment_update"
	MessageTypeCommentDelete MessageType = "comment_delete"
)

// WebSocket message structure
type WSMessage struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
	Room    string      `json:"room,omitempty"` // post_slug or blog_slug
}

// Client connection
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	rooms  map[string]bool // which rooms this client is subscribed to
	userID string          // for authentication
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
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			for room := range client.rooms {
				if h.rooms[room] == nil {
					h.rooms[room] = make(map[*Client]bool)
				}
				h.rooms[room][client] = true
			}
			h.mutex.Unlock()
			log.Printf("Client connected to rooms: %v", client.rooms)

		case client := <-h.unregister:
			h.mutex.Lock()
			for room := range client.rooms {
				if clients, ok := h.rooms[room]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						close(client.send)
						if len(clients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			h.mutex.Unlock()
			log.Printf("Client disconnected from rooms: %v", client.rooms)

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
						delete(clients, client)
						close(client.send)
					}
				}
			}
			h.mutex.RUnlock()
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
	default:
		log.Printf("Failed to broadcast message to room %s", room)
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
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    h,
		rooms:  rooms,
		userID: userID,
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Handle writing messages to WebSocket
func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// Handle reading messages from WebSocket (for room subscription changes)
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var msg struct {
			Action string   `json:"action"`
			Rooms  []string `json:"rooms"`
		}

		err := c.conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// Handle room subscription changes
		if msg.Action == "subscribe" {
			c.hub.mutex.Lock()
			for _, room := range msg.Rooms {
				c.rooms[room] = true
				if c.hub.rooms[room] == nil {
					c.hub.rooms[room] = make(map[*Client]bool)
				}
				c.hub.rooms[room][c] = true
			}
			c.hub.mutex.Unlock()
		} else if msg.Action == "unsubscribe" {
			c.hub.mutex.Lock()
			for _, room := range msg.Rooms {
				delete(c.rooms, room)
				if clients, ok := c.hub.rooms[room]; ok {
					delete(clients, c)
				}
			}
			c.hub.mutex.Unlock()
		}
	}
}
