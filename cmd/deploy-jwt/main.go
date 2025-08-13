package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"tomlord.io-backend/internal/config"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	GoogleID string `json:"google_id"`
	jwt.RegisteredClaims
}

func main() {
	config.Load()

	jwtSecret := viper.GetString("SYNC_SESSION_SECRET")
	fmt.Println("SYNC_SESSION_SECRET:", jwtSecret)
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
	}

	claims := Claims{
		UserID:   "deployment-service-user",
		Email:    "deploy@tomlord.io",
		Name:     "Deployment Service",
		GoogleID: "deployment-service",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: nil,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tomlord.io-backend",
			Subject:   "deployment-service-user",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(err)
	}
	fmt.Println("🚀 Deployment JWT Token (10 years validity):")
	fmt.Println(tokenString)
	fmt.Println("")
	fmt.Println("📝 Usage:")
	fmt.Println("1. Set this token as Vercel environment variable AUTH_TOKEN")
	fmt.Println("2. The token will be automatically used to sync blogs during deployment")
	fmt.Println("")
	fmt.Printf("🧪 Test command:\ncurl -H \"Authorization: Bearer %s\" \\\n     -X POST http://localhost:8080/api/sync-blogs \\\n     -d '[{\n       \"title\": \"Test Article\",\n       \"slug\": \"test-post\",\n       \"date\": \"2025-01-29\",\n       \"lang\": \"zh-tw\",\n       \"duration\": \"2min\",\n       \"tags\": [\"test\"],\n       \"description\": \"This is a test article\",\n       \"is_published\": true\n     }]'\n", tokenString)
}
