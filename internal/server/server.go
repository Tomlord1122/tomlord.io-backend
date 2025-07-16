package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

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
	authMiddleware *middleware.AuthMiddleware
	wsHub          *websocket.Hub
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080
	}

	// Initialize database service
	ctx := context.Background()
	dbService, err := database.NewDBService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	authService := auth.NewAuthService(dbService)
	messageService := services.NewMessageService(dbService)
	blogService := services.NewBlogService(dbService)

	// Initialize auth middleware with JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // Default secret - change in production
	}
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

func (s *Server) gracefulShutdown(ctx context.Context, server *http.Server) error {
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
