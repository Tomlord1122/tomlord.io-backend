package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
	db "tomlord.io-backend/internal/db_sqlc"
)

// DBService represents a service that interacts with a database using sqlc generated code.
type DBService interface {
	// Health returns a map of health status information.
	Health() map[string]string

	// Close terminates the database connection.
	Close()

	// GetQueries returns the sqlc generated queries
	GetQueries() *db.Queries

	// GetPool returns the connection pool
	GetPool() *pgxpool.Pool

	// WithTx executes the provided function within a database transaction.
	// The transaction is committed if fn returns nil, otherwise it is rolled back.
	RunTransaction(ctx context.Context, fn func(q *db.Queries) error) error
}

type dbService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// Database connection settings (migrated from legacy database.go)
var (
	database string
	password string
	username string
	port     string
	host     string
	schema   string
)

// NewDBService creates a new database service with connection pool
func NewDBService(ctx context.Context) (DBService, error) {
	var connStr string

	// Cache envs into module variables for logging and Close()
	database = viper.GetString("BLUEPRINT_DB_DATABASE")
	password = viper.GetString("BLUEPRINT_DB_PASSWORD")
	username = viper.GetString("BLUEPRINT_DB_USERNAME")
	port = viper.GetString("BLUEPRINT_DB_PORT")
	host = viper.GetString("BLUEPRINT_DB_HOST")
	schema = viper.GetString("BLUEPRINT_DB_SCHEMA")

	if databaseURL := viper.GetString("DATABASE_URL"); databaseURL != "" {
		connStr = databaseURL
	} else {
		sslMode := "disable"
		if viper.GetString("APP_ENV") == "production" {
			sslMode = "require"
		}

		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s",
			username, password, host, port, database, sslMode, schema)
	}

	log.Printf("Connecting to database with SSL mode: %s", getSslModeFromConnStr(connStr))

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	queries := db.New(pool)

	return &dbService{
		pool:    pool,
		queries: queries,
	}, nil
}

// Helper function to extract SSL mode for logging
func getSslModeFromConnStr(connStr string) string {
	if len(connStr) > 0 {
		if viper.GetString("APP_ENV") == "production" {
			return "require"
		}
		return "disable"
	}
	return "unknown"
}

// Health checks the health of the database connection.
func (s *dbService) Health() map[string]string {
	ctx := context.Background()
	stats := make(map[string]string)

	// Ping the database
	err := s.pool.Ping(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("Database health check failed: %v", err)
		return stats
	}

	// Database is up
	stats["status"] = "up"
	stats["message"] = "Database is healthy"

	// Get pool stats
	poolStats := s.pool.Stat()
	stats["acquired_connections"] = fmt.Sprintf("%d", poolStats.AcquiredConns())
	stats["idle_connections"] = fmt.Sprintf("%d", poolStats.IdleConns())
	stats["max_connections"] = fmt.Sprintf("%d", poolStats.MaxConns())
	stats["total_connections"] = fmt.Sprintf("%d", poolStats.TotalConns())

	return stats
}

// Close closes the database connection pool.
func (s *dbService) Close() {
	if s.pool != nil {
		s.pool.Close()
		log.Printf("Disconnected from database: %s", database)
	}
}

// GetQueries returns the sqlc generated queries
func (s *dbService) GetQueries() *db.Queries {
	return s.queries
}

// GetPool returns the connection pool
func (s *dbService) GetPool() *pgxpool.Pool {
	return s.pool
}

// WithTx executes the provided function within a transaction.
func (s *dbService) RunTransaction(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}

	qtx := s.queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
