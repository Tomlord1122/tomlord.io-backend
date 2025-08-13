package originpolicy

import (
	"strings"

	"github.com/spf13/viper"
)

// AllowedOrigins centralizes the origin policy for both HTTP CORS and WebSocket
func AllowedOrigins() []string {
	appEnv := viper.GetString("APP_ENV")
	if appEnv == "production" {
		// Prefer explicit configuration
		if allowed := strings.TrimSpace(viper.GetString("ALLOWED_ORIGINS")); allowed != "" {
			return splitAndTrim(allowed)
		}
		if frontend := strings.TrimSpace(viper.GetString("FRONTEND_URL")); frontend != "" {
			return []string{frontend}
		}
		// Sensible production default
		return []string{"https://tomlord.fyi"}
	}

	// Development defaults
	// Allow overriding with ALLOWED_ORIGINS
	if allowed := strings.TrimSpace(viper.GetString("ALLOWED_ORIGINS")); allowed != "" {
		return splitAndTrim(allowed)
	}

	// Minimal localhost list for development
	return []string{
		"http://localhost:5173",
	}
}

func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
