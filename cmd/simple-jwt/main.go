package main

import (
	"fmt"
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
	// 服務器默認使用的密鑰（如果環境變數為空）
	jwtSecret := "your-secret-key-change-in-production"

	// 創建測試用戶的 claims
	claims := Claims{
		UserID:   "550e8400-e29b-41d4-a716-446655440000", // 測試 UUID
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

	fmt.Println("🔐 測試 JWT Token:")
	fmt.Println(tokenString)
	fmt.Println("")
	fmt.Println("🧪 測試命令:")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:8080/auth/me\n", tokenString)
	fmt.Println("")
	fmt.Println("📋 Postman 使用:")
	fmt.Println("1. Authorization > Bearer Token")
	fmt.Println("2. 貼上上面的 token")
	fmt.Println("3. 測試 GET http://localhost:8080/auth/me")
}
