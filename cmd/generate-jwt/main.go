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
	// 使用與後端相同的 JWT_SECRET
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // 默認值與服務器一致
	}

	// 創建測試用戶的 claims
	claims := Claims{
		UserID:   "550e8400-e29b-41d4-a716-446655440000", // 測試用的 UUID
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

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(err)
	}

	fmt.Println("=== JWT Token 測試工具 ===")
	fmt.Println("")
	fmt.Println("Generated JWT Token:")
	fmt.Println(tokenString)
	fmt.Println("")
	fmt.Println("📋 複製上面的 token 到 Postman：")
	fmt.Println("1. 在 Authorization tab 選擇 'Bearer Token'")
	fmt.Println("2. 貼上這個 token")
	fmt.Println("3. 測試 GET /auth/me")
	fmt.Println("")
	fmt.Println("🔍 Token 內容：")
	fmt.Printf("  User ID: %s\n", claims.UserID)
	fmt.Printf("  Email: %s\n", claims.Email)
	fmt.Printf("  Name: %s\n", claims.Name)
	fmt.Printf("  Google ID: %s\n", claims.GoogleID)
	fmt.Printf("  過期時間: %s\n", claims.ExpiresAt.Time.Format("2006-01-02 15:04:05"))
}
