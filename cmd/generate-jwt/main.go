package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	GoogleID string `json:"google_id"`
	jwt.RegisteredClaims
}

func main() {
	// Use the same JWT_SECRET as the backend
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // default secret
	}

	// Create claims for the testing user
	claims := Claims{
		UserID:   "550e8400-e29b-41d4-a716-446655440000", // testing user UUID
		Email:    "test@example.com",
		Name:     "Test User",
		GoogleID: "test-google-id-12345",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tomlord.io-backend",
			Subject:   "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(err)
	}

	fmt.Println("=== JWT Token Testing Tool ===")
	fmt.Println("")
	fmt.Println("Generated JWT Token:")
	fmt.Println(tokenString)
	fmt.Println("")
	fmt.Println("📋 Copy the token above to Postman:")
	fmt.Println("1. Select 'Bearer Token' in the Authorization tab")
	fmt.Println("2. Paste the token")
	fmt.Println("3. Test GET /auth/me")
	fmt.Println("")
	fmt.Println("🔍 Token content:")
	fmt.Printf("  User ID: %s\n", claims.UserID)
	fmt.Printf("  Email: %s\n", claims.Email)
	fmt.Printf("  Name: %s\n", claims.Name)
	fmt.Printf("  Google ID: %s\n", claims.GoogleID)
	fmt.Printf("  Expiration time: %s\n", claims.ExpiresAt.Time.Format("2006-01-02 15:04:05"))
}
