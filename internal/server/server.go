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
)

type Server struct {
	port           int
	dbService      database.DBService
	authService    *auth.AuthService
	messageService *services.MessageService
	authMiddleware *middleware.AuthMiddleware
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080 // default port
	}

	// Initialize database service
	ctx := context.Background()
	dbService, err := database.NewDBService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	// Initialize auth service
	authService := auth.NewAuthService(dbService)

	// Initialize message service
	messageService := services.NewMessageService(dbService)

	// Initialize auth middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // Default secret - change in production
	}
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret, authService)

	newServer := &Server{
		port:           port,
		dbService:      dbService,
		authService:    authService,
		messageService: messageService,
		authMiddleware: authMiddleware,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", newServer.port),
		Handler:      newServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server, nil
}

// Close gracefully closes the server resources
func (s *Server) Close() {
	if s.dbService != nil {
		s.dbService.Close()
	}
}
