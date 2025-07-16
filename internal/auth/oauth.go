package auth

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/spf13/viper"
	"tomlord.io-backend/internal/database"
	"tomlord.io-backend/internal/db"
)

type AuthService struct {
	dbService database.DBService
}

type UserInfo struct {
	ID         string `json:"id"`
	GoogleID   string `json:"google_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	PictureURL string `json:"picture_url"`
}

// NewAuthService creates a new auth service
func NewAuthService(dbService database.DBService) *AuthService {
	// Initialize Goth providers
	setupOAuthProviders()

	return &AuthService{
		dbService: dbService,
	}
}

// setupOAuthProviders configures OAuth providers
func setupOAuthProviders() {
	// Load configuration
	viper.AutomaticEnv()

	googleClientID := viper.GetString("GOOGLE_CLIENT_ID")
	googleClientSecret := viper.GetString("GOOGLE_CLIENT_SECRET")
	callbackURL := viper.GetString("GOOGLE_CALLBACK_URL")

	if googleClientID == "" {
		googleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if googleClientSecret == "" {
		googleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	if callbackURL == "" {
		callbackURL = os.Getenv("GOOGLE_CALLBACK_URL")
		if callbackURL == "" {
			callbackURL = "http://localhost:8080/auth/google/callback"
		}
	}

	if googleClientID == "" || googleClientSecret == "" {
		log.Fatal("Google OAuth credentials not found in environment variables")
	}

	// Setup session store
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "your-session-secret-change-in-production" // Default secret
		log.Println("Warning: Using default session secret. Set SESSION_SECRET in production!")
	}

	// Create session store
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: 2,     // Lax
	}

	// Configure gothic to use our session store
	gothic.Store = store

	goth.UseProviders(
		google.New(googleClientID, googleClientSecret, callbackURL, "email", "profile"),
	)

	log.Println("OAuth providers configured successfully")
}

// CreateOrUpdateUser creates a new user or updates existing user from OAuth data
func (a *AuthService) CreateOrUpdateUser(ctx context.Context, gothUser goth.User) (*UserInfo, error) {
	queries := a.dbService.GetQueries()

	// Try to get existing user by Google ID
	_, err := queries.GetUserByGoogleID(ctx, gothUser.UserID)
	if err != nil {
		// User doesn't exist, create new one
		pictureURL := pgtype.Text{}
		if gothUser.AvatarURL != "" {
			if err := pictureURL.Scan(gothUser.AvatarURL); err != nil {
				return nil, fmt.Errorf("failed to scan picture URL: %w", err)
			}
		}

		user, err := queries.CreateUser(ctx, db.CreateUserParams{
			GoogleID:   gothUser.UserID,
			Email:      gothUser.Email,
			Name:       gothUser.Name,
			PictureUrl: pictureURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		return &UserInfo{
			ID:         uuid.UUID(user.ID.Bytes).String(),
			GoogleID:   user.GoogleID,
			Email:      user.Email,
			Name:       user.Name,
			PictureURL: user.PictureUrl.String,
		}, nil
	}

	// User exists, update their information and return
	pictureURL := pgtype.Text{}
	if gothUser.AvatarURL != "" {
		if err := pictureURL.Scan(gothUser.AvatarURL); err != nil {
			return nil, fmt.Errorf("failed to scan picture URL: %w", err)
		}
	}

	updatedUser, err := queries.UpdateUserByGoogleID(ctx, db.UpdateUserByGoogleIDParams{
		GoogleID:   gothUser.UserID,
		Email:      gothUser.Email,
		Name:       gothUser.Name,
		PictureUrl: pictureURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &UserInfo{
		ID:         uuid.UUID(updatedUser.ID.Bytes).String(),
		GoogleID:   updatedUser.GoogleID,
		Email:      updatedUser.Email,
		Name:       updatedUser.Name,
		PictureURL: updatedUser.PictureUrl.String,
	}, nil
}

// GetUserByID retrieves a user by their ID
func (a *AuthService) GetUserByID(ctx context.Context, userID string) (*UserInfo, error) {
	queries := a.dbService.GetQueries()

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := queries.GetUserByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &UserInfo{
		ID:         uuid.UUID(user.ID.Bytes).String(),
		GoogleID:   user.GoogleID,
		Email:      user.Email,
		Name:       user.Name,
		PictureURL: user.PictureUrl.String,
	}, nil
}
