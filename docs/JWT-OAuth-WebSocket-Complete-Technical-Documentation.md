# JWT, OAuth & WebSocket Complete Technical Documentation

## Table of Contents
1. [JWT (JSON Web Token) Principles and Implementation](#jwt-json-web-token-principles-and-implementation)
2. [OAuth 2.0 Principles and Implementation](#oauth-20-principles-and-implementation)
3. [WebSocket Principles and Implementation](#websocket-principles-and-implementation)
4. [System Integration Architecture](#system-integration-architecture)

---

## JWT (JSON Web Token) Principles and Implementation

### JWT Basic Principles

JWT is an open standard (RFC 7519) for securely transmitting information between parties. It consists of three parts:

```
Header.Payload.Signature
```

#### 1. Header
Contains the token type and signing algorithm used:
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

#### 2. Payload
Contains claims, which are statements about an entity (typically, the user) and additional data:
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "user@example.com",
  "name": "John Doe",
  "google_id": "google_user_id",
  "exp": 1735689600,
  "iat": 1735686000,
  "iss": "tomlord.io-backend",
  "sub": "123e4567-e89b-12d3-a456-426614174000"
}
```

#### 3. Signature
Used to verify that the message wasn't changed along the way:
```
HMACSHA256(
  base64UrlEncode(header) + "." +
  base64UrlEncode(payload),
  secret
)
```

### JWT Implementation Details

#### Claims Structure Definition

```go
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Name     string `json:"name"`
    GoogleID string `json:"google_id"`
    jwt.RegisteredClaims
}
```

#### JWT Generation Implementation

```go
func (a *AuthMiddleware) GenerateJWT(userInfo *auth.UserInfo) (string, error) {
    claims := Claims{
        UserID:   userInfo.ID,
        Email:    userInfo.Email,
        Name:     userInfo.Name,
        GoogleID: userInfo.GoogleID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // 1 hour expiry
            IssuedAt:  jwt.NewNumericDate(time.Now()),                    // Issue time
            NotBefore: jwt.NewNumericDate(time.Now()),                    // Valid from
            Issuer:    "tomlord.io-backend",                              // Issuer
            Subject:   userInfo.ID,                                       // Subject (user ID)
        },
    }

    // Create token using HMAC SHA256 algorithm
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(a.jwtSecret))
}
```

**Key Design Decisions:**
- **Expiry Time:** 1 hour, balancing security with user experience
- **Signing Algorithm:** HS256, suitable for single application symmetric encryption
- **Claims Content:** Contains basic user information, avoids sensitive data

#### JWT Validation Implementation

```go
func (a *AuthMiddleware) ValidateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Verify signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, jwt.ErrSignatureInvalid
        }
        return []byte(a.jwtSecret), nil
    })

    if err != nil {
        return nil, err
    }

    // Check token validity and claims integrity
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}
```

**Validation Process:**
1. Parse token structure
2. Verify signing algorithm type
3. Verify signature using secret key
4. Check expiry time and other standard claims
5. Return parsed user information

#### Token Extraction Strategy

```go
func (a *AuthMiddleware) extractToken(c *gin.Context) string {
    // Check Authorization header first
    authHeader := c.GetHeader("Authorization")
    if authHeader != "" {
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) == 2 && parts[0] == "Bearer" {
            return parts[1]
        }
    }

    // Fallback to Cookie
    cookie, err := c.Cookie("auth_token")
    if err == nil {
        return cookie
    }

    return ""
}
```

**Multiple Authentication Strategy:**
- **Bearer Token:** Suitable for API calls and frontend applications
- **HTTP Cookie:** Provides additional security and convenience

### JWT Middleware Implementation

#### RequireAuth: Mandatory Authentication Middleware

```go
func (a *AuthMiddleware) RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := a.extractToken(c)
        if tokenString == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
            c.Abort()
            return
        }

        claims, err := a.ValidateJWT(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Inject user information into context
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_name", claims.Name)
        c.Set("google_id", claims.GoogleID)

        c.Next()
    }
}
```

#### OptionalAuth: Optional Authentication Middleware

```go
func (a *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := a.extractToken(c)
        if tokenString != "" {
            claims, err := a.ValidateJWT(tokenString)
            if err == nil {
                // Only inject user information when token is valid
                c.Set("user_id", claims.UserID)
                c.Set("user_email", claims.Email)
                c.Set("user_name", claims.Name)
                c.Set("google_id", claims.GoogleID)
            }
        }

        c.Next() // Continue processing regardless of authentication success
    }
}
```

#### Permission Control Middleware

```go
func (a *AuthMiddleware) RequireSuperUserOrOwner() gin.HandlerFunc {
    return func(c *gin.Context) {
        // First ensure user is authenticated
        tokenString := a.extractToken(c)
        if tokenString == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
            c.Abort()
            return
        }

        claims, err := a.ValidateJWT(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Inject user information
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_name", claims.Name)
        c.Set("google_id", claims.GoogleID)

        // Check super user privileges
        if a.IsSuperUser(c) {
            c.Set("is_super_user", true)
        } else {
            c.Set("is_super_user", false)
        }

        c.Next()
    }
}
```

#### Sync Token Validation

```go
func RequireSyncToken() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "No bearer token provided"})
            c.Abort()
            return
        }

        tokenString := strings.TrimSpace(authHeader[len("Bearer "):])
        secret := viper.GetString("SYNC_SESSION_SECRET")
        if secret == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Sync secret not configured"})
            c.Abort()
            return
        }

        // Use different secret to validate sync token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return []byte(secret), nil
        })
        
        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid sync token"})
            c.Abort()
            return
        }

        c.Set("is_sync_request", true)
        c.Next()
    }
}
```

### Cookie Management

```go
func (a *AuthMiddleware) SetAuthCookie(c *gin.Context, token string) {
    c.SetCookie(
        "auth_token",
        token,
        3600,  // 1 hour expiry
        "/",   // Path
        "",    // Domain (empty string means current domain)
        false, // Secure (should be true in production)
        true,  // HttpOnly (prevents XSS attacks)
    )
}

func (a *AuthMiddleware) ClearAuthCookie(c *gin.Context) {
    c.SetCookie(
        "auth_token",
        "",
        -1, // Expire immediately
        "/",
        "",
        false,
        true,
    )
}
```

### JWT Security Considerations

1. **Secret Management:** Use environment variables to store secrets, use strong secrets in production
2. **Expiry Time:** Balance security with user experience, 1 hour is a reasonable choice
3. **HTTPS:** Production environment must use HTTPS for transmission
4. **HttpOnly Cookie:** Prevents XSS attacks
5. **Signature Verification:** Always verify token integrity

---

## OAuth 2.0 Principles and Implementation

### OAuth 2.0 Basic Principles

OAuth 2.0 is an authorization framework that allows third-party applications to obtain limited access to HTTP services. In our system, we use Google OAuth 2.0 for user authentication.

#### OAuth 2.0 Flow Diagram

```
User        Application      Google OAuth      Database
 |             |              |              |
 |--Login Req->|              |              |
 |             |--Auth Req--->|              |
 |<--Redirect to Google-------|              |
 |             |              |              |
 |--User Auth->|              |              |
 |             |<--Auth Code---|              |
 |             |              |              |
 |             |--Exchange Token->|           |
 |             |<--Access Token---|           |
 |             |              |              |
 |             |--Get User Info->|            |
 |             |<--User Data-----|            |
 |             |              |              |
 |             |--Create/Update User-------->|
 |             |<--User Info-----------------|
 |             |              |              |
 |             |--Generate JWT->|             |
 |<--Login Success--|          |             |
```

### OAuth Implementation Details

#### Goth Configuration

```go
func setupOAuthProviders() {
    googleClientID := viper.GetString("GOOGLE_CLIENT_ID")
    googleClientSecret := viper.GetString("GOOGLE_CLIENT_SECRET")
    callbackURL := viper.GetString("GOOGLE_CALLBACK_URL")

    if googleClientID == "" || googleClientSecret == "" {
        log.Fatal("Google OAuth credentials not found in environment variables")
    }

    // Setup session store
    sessionSecret := viper.GetString("SESSION_SECRET")
    if sessionSecret == "" {
        sessionSecret = "your-session-secret-change-in-production"
        log.Println("Warning: Using default session secret. Set SESSION_SECRET in production!")
    }

    // Create cookie store
    store := sessions.NewCookieStore([]byte(sessionSecret))
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   3600, // 1 hour
        HttpOnly: true,
        Secure:   viper.GetString("APP_ENV") == "production", // Use HTTPS in production
        SameSite: 2, // Lax
    }

    // Configure gothic to use our session store
    gothic.Store = store

    // Register Google OAuth provider
    goth.UseProviders(
        google.New(googleClientID, googleClientSecret, callbackURL, "email", "profile"),
    )

    log.Println("OAuth providers configured successfully")
}
```

**Configuration Points:**
- **Scope:** Request "email" and "profile" permissions
- **Session Security:** HttpOnly and Secure settings
- **SameSite:** Set to Lax to balance security and functionality

#### OAuth Route Handling

##### 1. Authorization Request Handling

```go
func (s *Server) authHandler(c *gin.Context) {
    provider := c.Param("provider")
    req := gothic.GetContextWithProvider(c.Request, provider)
    // Always start new OAuth flow
    gothic.BeginAuthHandler(c.Writer, req)
}
```

**Flow Explanation:**
1. User visits `/auth/google`
2. System redirects to Google OAuth authorization page
3. User authorizes on Google page

##### 2. Callback Handling

```go
func (s *Server) authCallbackHandler(c *gin.Context) {
    provider := c.Param("provider")
    req := gothic.GetContextWithProvider(c.Request, provider)

    // Complete OAuth flow, get user information
    gothUser, err := gothic.CompleteUserAuth(c.Writer, req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Create or update user in database
    userInfo, err := s.authService.CreateOrUpdateUser(c.Request.Context(), gothUser)
    if err != nil {
        fmt.Printf("Error creating/updating user: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user", "details": err.Error()})
        return
    }

    // Generate JWT token
    token, err := s.authMiddleware.GenerateJWT(userInfo)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    // Set authentication cookie
    s.authMiddleware.SetAuthCookie(c, token)

    // Redirect to frontend with token in URL
    frontendURL := viper.GetString("FRONTEND_URL")
    c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/auth/callback?token="+token)
}
```

**Callback Flow Details:**
1. Google redirects user to `/auth/google/callback`
2. System exchanges authorization code for access token
3. Use access token to get user information
4. Create or update user record in database
5. Generate application JWT token
6. Set authentication cookie
7. Redirect to frontend application

#### User Data Management

```go
type UserInfo struct {
    ID         string `json:"id"`
    GoogleID   string `json:"google_id"`
    Email      string `json:"email"`
    Name       string `json:"name"`
    PictureURL string `json:"picture_url"`
}

func (a *AuthService) CreateOrUpdateUser(ctx context.Context, gothUser goth.User) (*UserInfo, error) {
    queries := a.dbService.GetQueries()

    // Try to get existing user by Google ID
    _, err := queries.GetUserByGoogleID(ctx, gothUser.UserID)
    if err != nil {
        // User doesn't exist, create new user
        pictureURL := pgtype.Text{}
        if gothUser.AvatarURL != "" {
            if err := pictureURL.Scan(gothUser.AvatarURL); err != nil {
                return nil, fmt.Errorf("failed to scan picture URL: %w", err)
            }
        }

        user, err := queries.CreateUser(ctx, db.CreateUserParams{
            GoogleID:   gothUser.UserID,
            Email:      gothUser.Email,
            Name:       gothUser.Name,
            PictureUrl: pictureURL,
        })
        if err != nil {
            return nil, fmt.Errorf("failed to create user: %w", err)
        }

        return &UserInfo{
            ID:         uuid.UUID(user.ID.Bytes).String(),
            GoogleID:   user.GoogleID,
            Email:      user.Email,
            Name:       user.Name,
            PictureURL: user.PictureUrl.String,
        }, nil
    }

    // User exists, update information
    pictureURL := pgtype.Text{}
    if gothUser.AvatarURL != "" {
        if err := pictureURL.Scan(gothUser.AvatarURL); err != nil {
            return nil, fmt.Errorf("failed to scan picture URL: %w", err)
        }
    }

    updatedUser, err := queries.UpdateUserByGoogleID(ctx, db.UpdateUserByGoogleIDParams{
        GoogleID:   gothUser.UserID,
        Email:      gothUser.Email,
        Name:       gothUser.Name,
        PictureUrl: pictureURL,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to update user: %w", err)
    }

    return &UserInfo{
        ID:         uuid.UUID(updatedUser.ID.Bytes).String(),
        GoogleID:   updatedUser.GoogleID,
        Email:      updatedUser.Email,
        Name:       updatedUser.Name,
        PictureURL: updatedUser.PictureUrl.String,
    }, nil
}
```

**User Data Management Strategy:**
- **Upsert Pattern:** Update if user exists, create if not
- **Google ID as Primary Key:** Use unique ID provided by Google
- **Data Synchronization:** Update user information on each login to ensure data freshness

#### Logout Handling

```go
func (s *Server) logoutHandler(c *gin.Context) {
    s.authMiddleware.ClearAuthCookie(c)
    c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
```

#### User Information Retrieval

```go
func (s *Server) getMeHandler(c *gin.Context) {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"user": user})
}
```

### OAuth Security Considerations

1. **HTTPS:** Production environment must use HTTPS
2. **State Parameter:** Goth automatically handles CSRF protection
3. **Session Security:** HttpOnly and Secure cookie settings
4. **Secret Management:** OAuth credentials stored in environment variables
5. **Scope Limitation:** Only request necessary permissions (email, profile)

---

## WebSocket Principles and Implementation

### WebSocket Basic Principles

WebSocket is a communication protocol that provides full-duplex communication channels between client and server. Unlike HTTP, WebSocket connections are persistent, allowing real-time bidirectional data exchange.

#### WebSocket Handshake Process

```
Client                    Server
  |                        |
  |--HTTP Upgrade Request->|
  |  Connection: Upgrade   |
  |  Upgrade: websocket    |
  |  Sec-WebSocket-Key     |
  |                        |
  |<--HTTP 101 Switching --|
  |   Protocols            |
  |   Connection: Upgrade  |
  |   Upgrade: websocket   |
  |                        |
  |<----WebSocket Connection---->|
  |                        |
```

### WebSocket Architecture Design

Our WebSocket system adopts a Hub-Client architecture:

```
                    Hub (Central Manager)
                        |
        +---------------+---------------+
        |               |               |
    Client A        Client B        Client C
        |               |               |
   [Room: blog-1]  [Room: blog-1]  [Room: blog-2]
```

#### Core Components

##### 1. Hub: Central Manager

```go
type Hub struct {
    // Room -> Client mapping
    rooms map[string]map[*Client]bool

    // Register/unregister client channels
    register   chan *Client
    unregister chan *Client

    // Broadcast message channel
    broadcast chan WSMessage

    // Thread-safe mutex
    mutex sync.RWMutex
}
```

**Hub Responsibilities:**
- Manage all client connections
- Handle room subscription/unsubscription
- Broadcast messages to specific rooms
- Clean up invalid connections

##### 2. Client: Client Connection

```go
type Client struct {
    conn     *websocket.Conn     // WebSocket connection
    send     chan []byte         // Send message buffer channel
    hub      *Hub               // Hub reference
    rooms    map[string]bool    // Subscribed rooms
    userID   string             // User ID (for authentication)
    lastPong time.Time          // Last heartbeat time
    mutex    sync.RWMutex       // Thread-safe mutex
}
```

**Client Features:**
- Each WebSocket connection corresponds to one Client
- Supports multi-room subscription
- Built-in heartbeat detection
- Thread-safe design

### WebSocket Implementation Details

#### Connection Upgrade and Authentication

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        for _, allowed := range originpolicy.AllowedOrigins() {
            if origin == allowed {
                return true
            }
        }
        log.Printf("WebSocket connection rejected from origin: %s", origin)
        return false
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID string) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade error: %v", err)
        return
    }

    // Get initial room subscriptions from query parameters
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

    // Start read/write goroutines
    go client.writePump()
    go client.readPump()
}
```

**Connection Establishment Flow:**
1. Check request origin (CORS policy)
2. Upgrade HTTP connection to WebSocket
3. Parse initial room subscriptions
4. Create Client instance
5. Register to Hub
6. Start read/write goroutines

#### Hub Running Logic

```go
func (h *Hub) Run() {
    // Start cleanup goroutine
    go h.cleanupStaleConnections()

    for {
        select {
        case client := <-h.register:
            // Client registration
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
            // Client unregistration
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

            // Close send channel
            select {
            case <-client.send:
            default:
                close(client.send)
            }

            h.mutex.Unlock()
            log.Printf("Client %s disconnected from rooms: %v", client.userID, client.rooms)

        case message := <-h.broadcast:
            // Broadcast message
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
                        // Client send channel blocked, remove the client
                        delete(clients, client)
                        close(client.send)
                    }
                }
            }
            h.mutex.RUnlock()
        }
    }
}
```

#### Heartbeat Detection Mechanism

##### Write Pump (writePump)

```go
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
```

##### Read Pump (readPump)

```go
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
```

**Heartbeat Mechanism Parameters:**
```go
const (
    writeWait = 10 * time.Second    // Write timeout
    pongWait = 60 * time.Second     // Pong response timeout
    pingPeriod = (pongWait * 9) / 10 // Ping send interval
    maxMessageSize = 512            // Maximum message size
)
```

#### Connection Cleanup Mechanism

```go
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
```

#### Message Broadcasting System

##### Message Type Definition

```go
type MessageType string

const (
    MessageTypeNewComment    MessageType = "new_comment"
    MessageTypeThumbUpdate   MessageType = "thumb_update"
    MessageTypeCommentUpdate MessageType = "comment_update"
    MessageTypeCommentDelete MessageType = "comment_delete"
    MessageTypePing          MessageType = "ping"
    MessageTypePong          MessageType = "pong"
)

type WSMessage struct {
    Type    MessageType `json:"type"`
    Payload interface{} `json:"payload"`
    Room    string      `json:"room,omitempty"` // post_slug or blog_slug
}
```

##### Broadcast Implementation

```go
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
```

#### Business Integration Examples

##### New Comment Broadcast

```go
func (s *Server) createMessageHandler(c *gin.Context) {
    // ... Create message logic ...

    message, err := s.messageService.CreateMessage(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
        return
    }

    // Broadcast new message to WebSocket clients
    room := req.PostSlug // Use PostSlug as room name
    s.wsHub.BroadcastToRoom(room, websocket.MessageTypeNewComment, message)

    c.JSON(http.StatusCreated, gin.H{"message": message})
}
```

##### Thumb Update Broadcast

```go
func (s *Server) toggleThumbHandler(c *gin.Context) {
    // ... Toggle thumb logic ...

    thumbed, err := s.messageService.ToggleMessageThumb(c.Request.Context(), messageID, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle thumb"})
        return
    }

    count, err := s.messageService.GetThumbCount(c.Request.Context(), messageID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get thumb count"})
        return
    }

    // Broadcast thumb update to WebSocket clients
    room := message.PostSlug
    thumbData := map[string]any{
        "message_id":  messageID,
        "thumbed":     thumbed,
        "thumb_count": count,
        "user_id":     userID,
    }

    s.wsHub.BroadcastToRoom(room, websocket.MessageTypeThumbUpdate, thumbData)

    c.JSON(http.StatusOK, gin.H{
        "thumbed":     thumbed,
        "thumb_count": count,
    })
}
```

### WebSocket Security and Performance Considerations

#### Security

1. **Origin Check:** Use CORS policy to restrict connection origins
2. **Authentication Integration:** Support optional authentication, inject user ID into connection
3. **Message Size Limit:** Prevent large message attacks
4. **Connection Limit:** Can be implemented through load balancer

#### Performance Optimization

1. **Buffered Channels:** Use buffered channels to avoid blocking
2. **Goroutine Pool:** Use independent goroutines for each connection
3. **Memory Management:** Clean up invalid connections promptly
4. **Message Serialization:** Use efficient JSON serialization

#### Scalability

1. **Room Mechanism:** Support multi-room isolated broadcasting
2. **Dynamic Subscription:** Clients can dynamically subscribe/unsubscribe from rooms
3. **Horizontal Scaling:** Can implement cross-server broadcasting through Redis

---

## System Integration Architecture

### Complete Authentication Flow

```
Frontend App            Backend Service         Google OAuth           Database
    |                      |                        |                    |
    |--Click Login Button->|                        |                    |
    |                      |--Redirect to Google--->|                    |
    |<--Redirect to Google--|                        |                    |
    |                      |                        |                    |
    |--User Authorization->|                        |                    |
    |                      |<--Authorization Code---|                    |
    |                      |                        |                    |
    |                      |--Exchange Access Token->|                   |
    |                      |<--User Data-------------|                   |
    |                      |                        |                    |
    |                      |--Create/Update User------------------------->|
    |                      |<--User Info-----------------------------|
    |                      |                        |                    |
    |                      |--Generate JWT Token-----|                    |
    |<--Redirect + JWT Token|                        |                    |
    |                      |                        |                    |
    |--API Request + JWT--->|                        |                    |
    |                      |--Validate JWT---------->|                    |
    |<--API Response-------|                        |                    |
```

### Real-time Communication Architecture

```
Frontend Client A      WebSocket Hub           Frontend Client B
     |                      |                      |
     |--Establish WS Conn--->|                      |
     |                      |<--Establish WS Conn--|
     |                      |                      |
     |--Subscribe "blog-1"-->|                      |
     |                      |<--Subscribe "blog-1"-|
     |                      |                      |
     |--Send Message-------->|                      |
     |                      |--Broadcast to "blog-1"->|
     |                      |                      |
     |<--Receive Message-----|                      |
     |                      |--Receive Message---->|
```

### Configuration Management

```go
// Environment variable configuration
func Load() {
    viper.AutomaticEnv()
    
    // Read .env file in development environment
    env := viper.GetString("APP_ENV")
    if env == "" || env == "local" {
        viper.SetConfigFile(".env")
        viper.SetConfigType("env")
        _ = viper.ReadInConfig()
    }

    // Set default values
    viper.SetDefault("APP_ENV", "local")
    viper.SetDefault("PORT", 8080)
    viper.SetDefault("JWT_SECRET", "your-secret-key-change-in-production")
    viper.SetDefault("FRONTEND_URL", "http://localhost:5173")
    viper.SetDefault("GOOGLE_CALLBACK_URL", "http://localhost:8080/auth/google/callback")
    viper.SetDefault("SESSION_SECRET", "your-session-secret-change-in-production")
    viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:5173")
    viper.SetDefault("SYNC_SESSION_SECRET", "your-sync-session-secret-change-in-production")
}
```

### Route Design

```go
func (s *Server) RegisterRoutes() http.Handler {
    r := gin.Default()

    // CORS configuration
    r.Use(SetupCORS())

    // WebSocket route (supports optional authentication)
    r.GET("/ws", s.authMiddleware.OptionalAuth(), s.websocketHandler)

    // Authentication routes
    auth := r.Group("/auth")
    {
        auth.GET("/:provider", s.authHandler)                                    // OAuth start
        auth.GET("/:provider/callback", s.authCallbackHandler)                   // OAuth callback
        auth.POST("/logout", s.logoutHandler)                                    // Logout
        auth.GET("/me", s.authMiddleware.RequireAuth(), s.getMeHandler)          // Get user info
    }

    // API routes
    api := r.Group("/api")
    {
        // Blog routes
        blogs := api.Group("/blogs")
        {
            blogs.GET("/", s.listBlogsHandler)                                   // Public: list
            blogs.GET("/:slug", s.getBlogBySlugHandler)                          // Public: details
            blogs.GET("/:slug/messages", s.authMiddleware.OptionalAuth(), s.getMessagesByBlogSlugHandler) // Optional auth: messages
        }

        // Sync routes (special authentication)
        api.POST("/sync-blogs", middleware.RequireSyncToken(), s.syncBlogsHandler)

        // Message routes
        messages := api.Group("/messages")
        {
            messages.POST("/", s.authMiddleware.RequireAuth(), s.createMessageHandler)                    // Requires auth: create
            messages.PUT("/:id", s.authMiddleware.RequireAuth(), s.updateMessageHandler)                 // Requires auth: update
            messages.DELETE("/:id", s.authMiddleware.RequireSuperUserOrOwner(), s.deleteMessageHandler)  // Requires permission: delete
            messages.POST("/:id/thumb", s.authMiddleware.RequireAuth(), s.toggleThumbHandler)            // Requires auth: thumb
        }
    }

    return r
}
```

### Error Handling and Logging

```go
// JWT validation error
if err != nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
    c.Abort()
    return
}

// OAuth error
if err != nil {
    fmt.Printf("Error creating/updating user: %v\n", err)
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": "Failed to process user", 
        "details": err.Error(),
    })
    return
}

// WebSocket error
if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
    log.Printf("WebSocket error for client %s: %v", c.userID, err)
}
```

### Production Environment Considerations

#### Security Checklist

- [ ] Use HTTPS (set `Secure: true` for cookies)
- [ ] Change default secrets (JWT_SECRET, SESSION_SECRET, SYNC_SESSION_SECRET)
- [ ] Set correct CORS origins
- [ ] Use environment variables to manage sensitive information
- [ ] Enable rate limiting
- [ ] Set appropriate cookie security attributes

#### Performance Optimization

- [ ] Set connection pool size
- [ ] Configure appropriate timeout values
- [ ] Implement connection limits
- [ ] Use load balancer
- [ ] Consider Redis for session storage

#### Monitoring and Logging

- [ ] Set up structured logging
- [ ] Monitor WebSocket connection count
- [ ] Track authentication success/failure rates
- [ ] Monitor API response times
- [ ] Set up alerting mechanisms

This complete technical documentation covers the principles, implementation, and best practices of JWT, OAuth, and WebSocket. Each section includes detailed code examples and explanations to help you understand how the entire system works.
