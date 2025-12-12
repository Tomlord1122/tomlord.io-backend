package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"tomlord.io-backend/internal/auth"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	GoogleID string `json:"google_id"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	jwtSecret   string
	authService *auth.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtSecret string, authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:   jwtSecret,
		authService: authService,
	}
}

// GenerateJWT generates a JWT token for a user
func (a *AuthMiddleware) GenerateJWT(userInfo *auth.UserInfo) (string, error) {
	claims := Claims{
		UserID:   userInfo.ID,
		Email:    userInfo.Email,
		Name:     userInfo.Name,
		GoogleID: userInfo.GoogleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // 1 hour
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tomlord.io-backend",
			Subject:   userInfo.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// ValidateJWT validates a JWT token and returns claims
func (a *AuthMiddleware) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(a.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RequireAuth middleware that requires authentication
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

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("google_id", claims.GoogleID)

		c.Next()
	}
}

// OptionalAuth middleware that optionally checks for authentication
func (a *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := a.extractToken(c)
		if tokenString != "" {
			claims, err := a.ValidateJWT(tokenString)
			if err == nil {
				// Add user info to context
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Name)
				c.Set("google_id", claims.GoogleID)
			}
		}

		c.Next()
	}
}

// extractToken extracts token from Authorization header or cookie
func (a *AuthMiddleware) extractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Try cookie
	cookie, err := c.Cookie("auth_token")
	if err == nil {
		return cookie
	}

	return ""
}

// SetAuthCookie sets the authentication cookie
func (a *AuthMiddleware) SetAuthCookie(c *gin.Context, token string) {
	c.SetCookie(
		"auth_token",
		token,
		3600, // 1 hour in seconds
		"/",
		"",    // domain - empty for same domain
		false, // secure - set to true in production with HTTPS
		true,  // httpOnly
	)
}

// ClearAuthCookie clears the authentication cookie
func (a *AuthMiddleware) ClearAuthCookie(c *gin.Context) {
	c.SetCookie(
		"auth_token",
		"",
		-1, // expire immediately
		"/",
		"",
		false,
		true,
	)
}

// GetCurrentUser gets the current user from context
func GetCurrentUser(c *gin.Context) *auth.UserInfo {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil
	}

	email, _ := c.Get("user_email")
	name, _ := c.Get("user_name")
	googleID, _ := c.Get("google_id")

	return &auth.UserInfo{
		ID:       userID.(string),
		Email:    email.(string),
		Name:     name.(string),
		GoogleID: googleID.(string),
	}
}

// GetCurrentUserID gets the current user ID from context
func GetCurrentUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// IsCurrentUserSuperUser checks if the current user has super user privileges
func IsCurrentUserSuperUser(c *gin.Context) bool {
	isSuperUser, exists := c.Get("is_super_user")
	if !exists {
		return false
	}
	return isSuperUser.(bool)
}

// IsSuperUser checks if the current user has super user privileges
func (a *AuthMiddleware) IsSuperUser(c *gin.Context) bool {
	email, exists := c.Get("user_email")
	if !exists {
		return false
	}

	// TODO:[PRODUCTION] Add super user email to environment variables for production
	superUserEmail := "r12944044@csie.ntu.edu.tw"
	return email.(string) == superUserEmail
}

// RequireSuperUserOrOwner middleware that requires either super user privileges or ownership
func (a *AuthMiddleware) RequireSuperUserOrOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
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

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("google_id", claims.GoogleID)

		// Check if user is super user
		if a.IsSuperUser(c) {
			c.Set("is_super_user", true)
		} else {
			c.Set("is_super_user", false)
		}

		c.Next()
	}
}

// RequireSuperUser middleware that requires super user privileges
func (a *AuthMiddleware) RequireSuperUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
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

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("google_id", claims.GoogleID)

		// Check if user is super user
		if !a.IsSuperUser(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Super user privileges required"})
			c.Abort()
			return
		}

		c.Set("is_super_user", true)
		c.Next()
	}
}

// RequireSyncToken validates Authorization Bearer token signed with SYNC_SESSION_SECRET only.
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

		// Mark request as sync-authorized (optional context flag)
		c.Set("is_sync_request", true)

		c.Next()
	}
}
