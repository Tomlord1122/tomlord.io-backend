package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	GoogleID string `json:"google_id"`
	jwt.RegisteredClaims
}

func main() {
	// 使用Viper加載配置
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath("../..")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用環境變量
			log.Println("Config file not found, using environment variables")
		} else {
			log.Printf("Error reading config file: %v", err)
		}
	}

	// 從Viper獲取JWT密鑰
	jwtSecret := viper.GetString("JWT_SECRET")
	fmt.Println("JWT_SECRET:", jwtSecret)
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
	}

	// 創建服務用戶的claims
	claims := Claims{
		UserID:   "deployment-service-user",
		Email:    "deploy@tomlord.io",
		Name:     "Deployment Service",
		GoogleID: "deployment-service",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: nil, // 永久有效，不設置過期時間
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tomlord.io-backend",
			Subject:   "deployment-service-user",
		},
	}

	// 生成token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		panic(err)
	}

	fmt.Println("🚀 部署用 JWT Token (10年有效期):")
	fmt.Println(tokenString)
	fmt.Println("")
	fmt.Println("📝 使用方法:")
	fmt.Println("1. 將此token設置為Vercel環境變量 AUTH_TOKEN")
	fmt.Println("2. 部署時會自動使用此token同步blogs")
	fmt.Println("")
	fmt.Printf("🧪 測試命令:\ncurl -H \"Authorization: Bearer %s\" \\\n     -X POST http://localhost:8080/api/sync-blogs \\\n     -d '[{\n       \"title\": \"測試文章\",\n       \"slug\": \"test-post\",\n       \"date\": \"2025-01-29\",\n       \"lang\": \"zh-tw\",\n       \"duration\": \"2min\",\n       \"tags\": [\"test\"],\n       \"description\": \"這是一個測試文章\",\n       \"is_published\": true\n     }]'\n", tokenString)
}
