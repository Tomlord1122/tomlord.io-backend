package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"tomlord.io-backend/internal/originpolicy"
)

// SetupCORS configures CORS using a centralized origin policy
func SetupCORS() gin.HandlerFunc {
	config := cors.DefaultConfig()

	// Use the same allowed origins as WebSocket to keep parity
	config.AllowOrigins = originpolicy.AllowedOrigins()

	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	config.AllowHeaders = []string{
		"Accept",
		"Authorization",
		"Content-Type",
		"X-Requested-With",
		"X-Visitor-Key",
		"Origin",
	}
	config.ExposeHeaders = []string{"Content-Length"}
	config.MaxAge = 600 // 10 minutes

	return cors.New(config)
}
