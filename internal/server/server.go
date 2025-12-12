package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"tomlord.io-backend/internal/auth"
	"tomlord.io-backend/internal/database"
	"tomlord.io-backend/internal/middleware"
	"tomlord.io-backend/internal/services"
	"tomlord.io-backend/internal/websocket"
)

type Server struct {
	port           int
	dbService      database.DBService
	authService    *auth.AuthService
	messageService *services.MessageService
	blogService    *services.BlogService
	pageService    *services.PageService
	authMiddleware *middleware.AuthMiddleware
	wsHub          *websocket.Hub
}

func NewServer() (*http.Server, error) {
	port := viper.GetInt("PORT")

	// Initialize database service
	ctx := context.Background()
	dbService, err := database.NewDBService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	authService := auth.NewAuthService(dbService)
	messageService := services.NewMessageService(dbService)
	blogService := services.NewBlogService(dbService)
	pageService := services.NewPageService(dbService)

	// Initialize auth middleware with JWT secret
	jwtSecret := viper.GetString("JWT_SECRET")
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret, authService)

	// Create WebSocket hub and start it
	wsHub := websocket.NewHub()
	go wsHub.Run()

	NewServer := &Server{
		port:           port,
		dbService:      dbService,
		authService:    authService,
		messageService: messageService,
		blogService:    blogService,
		pageService:    pageService,
		authMiddleware: authMiddleware,
		wsHub:          wsHub,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server, nil
}

func (s *Server) Health() map[string]string {
	return map[string]string{
		"status":   "up",
		"database": s.dbService.Health()["status"],
	}
}
