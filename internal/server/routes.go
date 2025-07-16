package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"tomlord.io-backend/internal/middleware"
	"tomlord.io-backend/internal/services"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"}, // Add your frontend URLs
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	// Basic routes
	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.GET("/:provider", s.authHandler)
		auth.GET("/:provider/callback", s.authCallbackHandler)
		auth.POST("/logout", s.logoutHandler)
		auth.GET("/me", s.authMiddleware.RequireAuth(), s.getMeHandler)
	}

	// API routes
	api := r.Group("/api")
	{
		// Message routes (comments)
		messages := api.Group("/messages")
		{
			// Public routes (with optional auth for checking user thumbs)
			messages.GET("/post/:slug", s.authMiddleware.OptionalAuth(), s.getMessagesByPostHandler)

			// Protected routes
			messages.POST("/", s.authMiddleware.RequireAuth(), s.createMessageHandler)
			messages.PUT("/:id", s.authMiddleware.RequireAuth(), s.updateMessageHandler)
			messages.DELETE("/:id", s.authMiddleware.RequireAuth(), s.deleteMessageHandler)
			messages.POST("/:id/thumb", s.authMiddleware.RequireAuth(), s.toggleThumbHandler)
		}
	}

	return r
}

// Basic handlers
func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World from tomlord.io backend"
	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.dbService.Health())
}

// Auth handlers
func (s *Server) authHandler(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "provider", provider))

	if gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request); err == nil {
		// User already authenticated, redirect or return user info
		userInfo, err := s.authService.CreateOrUpdateUser(c.Request.Context(), gothUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user"})
			return
		}

		// Generate JWT token
		token, err := s.authMiddleware.GenerateJWT(userInfo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Set cookie and return user info
		s.authMiddleware.SetAuthCookie(c, token)
		c.JSON(http.StatusOK, gin.H{
			"user":  userInfo,
			"token": token,
		})
	} else {
		// Start OAuth flow
		gothic.BeginAuthHandler(c.Writer, c.Request)
	}
}

func (s *Server) authCallbackHandler(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "provider", provider))

	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create or update user in database
	userInfo, err := s.authService.CreateOrUpdateUser(c.Request.Context(), gothUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user"})
		return
	}

	// Generate JWT token
	token, err := s.authMiddleware.GenerateJWT(userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set cookie
	s.authMiddleware.SetAuthCookie(c, token)

	// Redirect to frontend or return JSON
	frontendURL := "http://localhost:5173" // Change this to your frontend URL
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"?auth=success")
}

func (s *Server) logoutHandler(c *gin.Context) {
	s.authMiddleware.ClearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (s *Server) getMeHandler(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Message handlers
func (s *Server) getMessagesByPostHandler(c *gin.Context) {
	postSlug := c.Param("slug")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Get current user ID if authenticated (for checking thumbs)
	userID := ""
	if currentUserID, exists := middleware.GetCurrentUserID(c); exists {
		userID = currentUserID
	}

	req := services.ListMessagesRequest{
		PostSlug: postSlug,
		Limit:    int32(limit),
		Offset:   int32(offset),
		UserID:   userID,
	}

	messages, err := s.messageService.GetMessagesByPostSlug(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (s *Server) createMessageHandler(c *gin.Context) {
	var req services.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	req.UserID = userID

	message, err := s.messageService.CreateMessage(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

func (s *Server) updateMessageHandler(c *gin.Context) {
	messageID := c.Param("id")

	var req services.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	req.MessageID = messageID
	req.UserID = userID

	message, err := s.messageService.UpdateMessage(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

func (s *Server) deleteMessageHandler(c *gin.Context) {
	messageID := c.Param("id")

	// Get current user ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err := s.messageService.DeleteMessage(c.Request.Context(), messageID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

func (s *Server) toggleThumbHandler(c *gin.Context) {
	messageID := c.Param("id")

	// Get current user ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	thumbed, err := s.messageService.ToggleMessageThumb(c.Request.Context(), messageID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle thumb"})
		return
	}

	// Get updated thumb count
	count, err := s.messageService.GetThumbCount(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get thumb count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"thumbed":     thumbed,
		"thumb_count": count,
	})
}
