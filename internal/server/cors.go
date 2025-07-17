package server

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupCORS configures CORS based on environment
func SetupCORS() gin.HandlerFunc {
	config := cors.DefaultConfig()

	// Get environment variables
	appEnv := os.Getenv("APP_ENV")
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	frontendURL := os.Getenv("FRONTEND_URL")

	if appEnv == "production" {
		// Production CORS configuration
		if allowedOrigins != "" {
			config.AllowOrigins = strings.Split(allowedOrigins, ",")
		} else if frontendURL != "" {
			// Fallback to FRONTEND_URL if ALLOWED_ORIGINS is not set
			config.AllowOrigins = []string{frontendURL}
		} else {
			// Default fallback for production
			config.AllowOrigins = []string{
				"https://tomlord.fyi",
			}
		}

		// Secure configuration for production
		config.AllowCredentials = true
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
		config.AllowHeaders = []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Requested-With",
			"Origin",
		}
		config.ExposeHeaders = []string{"Content-Length"}
		config.MaxAge = 43200 // 12 hours

	} else if appEnv == "minikube" {
		// Minikube CORS configuration
		config.AllowOrigins = []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://localhost:4173", // Vite preview
			"http://minikube.local",
			"http://192.168.49.2", // Minikube default IP
			"http://192.168.49.1", // Minikube alternative IP
		}
		config.AllowCredentials = true
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
		config.AllowHeaders = []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Requested-With",
			"Origin",
		}
		config.ExposeHeaders = []string{"Content-Length"}
		config.MaxAge = 43200 // 12 hours

	} else {
		// Development CORS configuration (more permissive)
		config.AllowOrigins = []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://localhost:4173", // Vite preview
		}
		config.AllowCredentials = true
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
		config.AllowHeaders = []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Requested-With",
			"Origin",
		}
	}

	return cors.New(config)
}
