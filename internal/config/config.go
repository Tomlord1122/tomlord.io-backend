package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Load initializes configuration for the application.
// It enables environment variables, loads local .env when APP_ENV is empty or "local",
// and sets sane defaults for required keys.
func Load() {
	// Map env var keys like FRONTEND_URL to viper keys like frontend.url if needed
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	// Priority: viper.Set() -> AutomaticEnv() -> .env -> SetDefault()
	// Local development: read from .env (ignore error if missing)
	env := viper.GetString("APP_ENV")
	if env == "" || env == "local" {
		viper.SetConfigFile(".env")
		viper.SetConfigType("env")
		_ = viper.ReadInConfig()
	}

	// Defaults
	viper.SetDefault("APP_ENV", "local")
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("JWT_SECRET", "your-secret-key-change-in-production") // TODO:[PRODUCTION]
	viper.SetDefault("FRONTEND_URL", "http://localhost:5173")
	viper.SetDefault("GOOGLE_CALLBACK_URL", "http://localhost:8080/auth/google/callback")
	viper.SetDefault("SESSION_SECRET", "your-session-secret-change-in-production") // TODO:[PRODUCTION]
	viper.SetDefault("BLUEPRINT_DB_SCHEMA", "public")
}
