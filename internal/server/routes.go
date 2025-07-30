package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"tomlord.io-backend/internal/middleware"
	"tomlord.io-backend/internal/services"
	"tomlord.io-backend/internal/websocket"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// CORS configuration - use the centralized SetupCORS function
	r.Use(SetupCORS())

	// Basic routes
	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)
	r.GET("/debug/jwt", s.debugJWTHandler)

	// WebSocket route
	r.GET("/ws", s.authMiddleware.OptionalAuth(), s.websocketHandler)

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
		// Blog routes
		blogs := api.Group("/blogs")
		{
			// Public routes
			blogs.GET("/", s.listBlogsHandler)
			blogs.GET("/:slug", s.getBlogBySlugHandler)
			blogs.GET("/:slug/messages", s.authMiddleware.OptionalAuth(), s.getMessagesByBlogSlugHandler)

			// Protected routes (for blog management - might want to add admin middleware later)
			isProduction := os.Getenv("APP_ENV") == "production"
			if !isProduction {
				blogs.POST("/", s.authMiddleware.RequireAuth(), s.createBlogHandler)
				blogs.PUT("/:slug", s.authMiddleware.RequireAuth(), s.updateBlogHandler)
				blogs.DELETE("/:slug", s.authMiddleware.RequireAuth(), s.deleteBlogHandler)
			}
		}

		// Temporary sync endpoint for initial blog migration (remove after sync)
		api.POST("/sync-blogs", s.authMiddleware.RequireAuth(), s.syncBlogsHandler)

		// Message routes (comments)
		messages := api.Group("/messages")
		{
			// Public routes (with optional auth for checking user thumbs)
			messages.GET("/post/:slug", s.authMiddleware.OptionalAuth(), s.getMessagesByPostHandler)
			messages.GET("/blog/:slug", s.authMiddleware.OptionalAuth(), s.getMessagesByBlogSlugHandler)

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

func (s *Server) debugJWTHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")

	response := gin.H{
		"auth_header": authHeader,
	}

	if authHeader != "" {
		// 嘗試驗證 token
		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		response["token_extracted"] = tokenString

		if tokenString != "" {
			claims, err := s.authMiddleware.ValidateJWT(tokenString)
			if err != nil {
				response["jwt_error"] = err.Error()
			} else {
				response["jwt_claims"] = claims
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// Auth handlers
func (s *Server) authHandler(c *gin.Context) {
	provider := c.Param("provider")
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "provider", provider))

	// Always start OAuth flow for new authentication
	gothic.BeginAuthHandler(c.Writer, c.Request)
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
		// Log the detailed error for debugging
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

	// Set cookie
	s.authMiddleware.SetAuthCookie(c, token)

	// Redirect to frontend auth callback with token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		// Fallback to default development URL
		frontendURL = "http://localhost:5173"
	}
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/auth/callback?token="+token)
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

	// Broadcast new comment to WebSocket clients
	room := req.PostSlug // Always use PostSlug as the room name for consistency
	s.wsHub.BroadcastToRoom(room, websocket.MessageTypeNewComment, message)

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

	// Get message info first to determine the room for broadcasting
	message, err := s.messageService.GetMessageByID(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	err = s.messageService.DeleteMessage(c.Request.Context(), messageID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	// Broadcast delete event to WebSocket clients
	room := message.PostSlug // Always use PostSlug as the room name for consistency

	deleteData := map[string]interface{}{
		"message_id": messageID,
		"user_id":    userID,
	}

	// Debug logging
	fmt.Printf("Broadcasting delete event to room '%s': %+v\n", room, deleteData)
	s.wsHub.BroadcastToRoom(room, websocket.MessageTypeCommentDelete, deleteData)

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

	// Get message info first to determine the room for broadcasting
	message, err := s.messageService.GetMessageByID(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
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

	// Broadcast thumb update to WebSocket clients
	room := message.PostSlug // Always use PostSlug as the room name for consistency

	thumbData := map[string]interface{}{
		"message_id":  messageID,
		"thumbed":     thumbed,
		"thumb_count": count,
		"user_id":     userID,
	}

	// Debug logging
	fmt.Printf("Broadcasting thumb update to room '%s': %+v\n", room, thumbData)
	s.wsHub.BroadcastToRoom(room, websocket.MessageTypeThumbUpdate, thumbData)

	c.JSON(http.StatusOK, gin.H{
		"thumbed":     thumbed,
		"thumb_count": count,
	})
}

// Blog handlers
func (s *Server) listBlogsHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	tag := c.Query("tag")
	lang := c.Query("lang")
	publishedOnly := c.DefaultQuery("published", "true") == "true"

	req := services.ListBlogsRequest{
		Limit:         int32(limit),
		Offset:        int32(offset),
		Tag:           tag,
		Lang:          lang,
		PublishedOnly: publishedOnly,
	}

	blogs, err := s.blogService.ListBlogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blogs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blogs": blogs})
}

func (s *Server) getBlogBySlugHandler(c *gin.Context) {
	slug := c.Param("slug")

	blog, err := s.blogService.GetBlogWithMessageCountBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blog": blog})
}

func (s *Server) createBlogHandler(c *gin.Context) {
	var req services.CreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blog, err := s.blogService.CreateBlog(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create blog"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"blog": blog})
}

func (s *Server) updateBlogHandler(c *gin.Context) {
	slug := c.Param("slug")

	var req services.UpdateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blog, err := s.blogService.UpdateBlogBySlug(c.Request.Context(), slug, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blog": blog})
}

func (s *Server) deleteBlogHandler(c *gin.Context) {
	slug := c.Param("slug")

	err := s.blogService.DeleteBlogBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete blog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blog deleted successfully"})
}

// Temporary sync handler for blog migration
func (s *Server) syncBlogsHandler(c *gin.Context) {
	var blogs []services.CreateBlogRequest
	if err := c.ShouldBindJSON(&blogs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results := []interface{}{}
	errors := []string{}

	for _, blogReq := range blogs {
		// Check if blog already exists
		existingBlog, err := s.blogService.GetBlogBySlug(c.Request.Context(), blogReq.Slug)

		if err == nil && existingBlog != nil {
			// Blog exists, update it
			updateReq := services.UpdateBlogRequest{
				Title:       blogReq.Title,
				Date:        blogReq.Date,
				Lang:        blogReq.Lang,
				Duration:    blogReq.Duration,
				Tags:        blogReq.Tags,
				Description: blogReq.Description,
				IsPublished: blogReq.IsPublished,
			}

			blog, err := s.blogService.UpdateBlogBySlug(c.Request.Context(), blogReq.Slug, updateReq)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to update %s: %v", blogReq.Slug, err))
			} else {
				results = append(results, map[string]interface{}{
					"action": "updated",
					"blog":   blog,
				})
			}
		} else {
			// Blog doesn't exist, create it
			blog, err := s.blogService.CreateBlog(c.Request.Context(), blogReq)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create %s: %v", blogReq.Slug, err))
			} else {
				results = append(results, map[string]interface{}{
					"action": "created",
					"blog":   blog,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"errors":  errors,
		"summary": gin.H{
			"total":   len(blogs),
			"success": len(results),
			"failed":  len(errors),
		},
	})
}

func (s *Server) getMessagesByBlogSlugHandler(c *gin.Context) {
	blogSlug := c.Param("slug")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Get current user ID if authenticated (for checking thumbs)
	userID := ""
	if currentUserID, exists := middleware.GetCurrentUserID(c); exists {
		userID = currentUserID
	}

	req := services.ListMessagesRequest{
		BlogSlug: blogSlug,
		Limit:    int32(limit),
		Offset:   int32(offset),
		UserID:   userID,
	}

	messages, err := s.messageService.GetMessagesByBlogSlug(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// WebSocket handler
func (s *Server) websocketHandler(c *gin.Context) {
	userID := ""
	if currentUserID, exists := middleware.GetCurrentUserID(c); exists {
		userID = currentUserID
	}

	s.wsHub.HandleWebSocket(c.Writer, c.Request, userID)
}
