# Environment Variables Migration

## Overview

This document describes the migration from hardcoded production values to environment variables in the tomlord.io-backend project. All production-specific configurations are now managed through environment variables, making the application more flexible and secure.

## Changes Made

### 1. CORS Configuration (`internal/server/cors.go`)

**Before:**
```go
// TODO:[PRODUCTION] Update these with your actual production domains
config.AllowOrigins = []string{
    "https://tomlord.fyi",
}
```

**After:**
```go
// Get environment variables
appEnv := os.Getenv("APP_ENV")
allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
frontendURL := os.Getenv("FRONTEND_URL")

if appEnv == "production" {
    if allowedOrigins != "" {
        config.AllowOrigins = strings.Split(allowedOrigins, ",")
    } else if frontendURL != "" {
        config.AllowOrigins = []string{frontendURL}
    } else {
        // Default fallback
        config.AllowOrigins = []string{"https://tomlord.fyi"}
    }
}
```

### 2. WebSocket Origin Check (`internal/websocket/hub.go`)

**Before:**
```go
// TODO:[PRODUCTION] Update these with your actual production domains
allowedOrigins := []string{
    "https://tomlord.vercel.app",
    "https://www.tomlord.io",
}
```

**After:**
```go
// Get allowed origins from environment variables
allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
frontendURL := os.Getenv("FRONTEND_URL")

var allowedOrigins []string
if allowedOriginsEnv != "" {
    allowedOrigins = strings.Split(allowedOriginsEnv, ",")
} else if frontendURL != "" {
    allowedOrigins = []string{frontendURL}
} else {
    // Default fallback
    allowedOrigins = []string{
        "https://tomlord.vercel.app",
        "https://www.tomlord.io",
    }
}
```

### 3. Auth Callback Redirect (`internal/server/routes.go`)

**Before:**
```go
// TODO:[PRODUCTION] Update this based on environment
frontendURL := "http://localhost:5173" // TODO:[PRODUCTION] Change this to your production frontend URL
```

**After:**
```go
frontendURL := os.Getenv("FRONTEND_URL")
if frontendURL == "" {
    // Fallback to default development URL
    frontendURL = "http://localhost:5173"
}
```

## New Environment Variables

### `FRONTEND_URL`
- **Purpose**: The URL of your frontend application
- **Format**: Full URL (e.g., `https://yourdomain.com`)
- **Fallback**: `http://localhost:5173` (development default)
- **Usage**: Used for OAuth callback redirects and as fallback for CORS

### `ALLOWED_ORIGINS`
- **Purpose**: Comma-separated list of allowed origins for CORS and WebSocket connections
- **Format**: Comma-separated URLs (e.g., `https://yourdomain.com,https://www.yourdomain.com`)
- **Fallback**: Uses `FRONTEND_URL` if not set
- **Usage**: Controls which domains can access the API and WebSocket connections

## Environment Variable Priority

The application uses the following priority order for determining allowed origins:

1. **`ALLOWED_ORIGINS`** - Highest priority, comma-separated list
2. **`FRONTEND_URL`** - Fallback if `ALLOWED_ORIGINS` is not set
3. **Default values** - Hardcoded fallbacks for each environment

## Configuration Examples

### Development
```bash
APP_ENV=local
FRONTEND_URL=http://localhost:5173
ALLOWED_ORIGINS=
```

### Minikube
```bash
APP_ENV=minikube
FRONTEND_URL=http://localhost:5173
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000,http://localhost:4173,http://minikube.local,http://192.168.49.2,http://192.168.49.1
```

### Production
```bash
APP_ENV=production
FRONTEND_URL=https://yourdomain.com
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

## Updated Files

- ✅ `internal/server/cors.go` - CORS configuration now uses environment variables
- ✅ `internal/server/routes.go` - Auth callback redirect uses environment variables
- ✅ `internal/websocket/hub.go` - WebSocket origin check uses environment variables
- ✅ `.env.example` - Added new environment variables with documentation
- ✅ `docker-compose.yml` - Added new environment variables
- ✅ `k8s/configmap.yaml` - Updated with new environment variables
- ✅ `k8s/minikube/configmap.yaml` - Updated with new environment variables
- ✅ `DEPLOYMENT_CHECKLIST.md` - Updated to reflect completed changes

## Benefits

1. **Flexibility**: Easy to change domains without code changes
2. **Security**: No hardcoded production values in source code
3. **Environment-specific**: Different configurations for different environments
4. **Maintainability**: Centralized configuration management
5. **Deployment**: No need to modify code for different deployment environments

## Migration Notes

- All existing functionality is preserved
- Fallback values ensure backward compatibility
- No breaking changes to existing API endpoints
- Environment variables are optional with sensible defaults 