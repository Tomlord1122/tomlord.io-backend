

# tomlord.io-backend Development Guide

This guide documents how to develop, run, and deploy a Golang backend using this project’s stack: Gin (HTTP), Viper (config), sqlc + pgx (PostgreSQL), Gorilla WebSocket, Goth (OAuth), Docker/Compose, and Fly.io.

English | [繁體中文](./README.zh-tw.md)

## Project Architecture

![Project Architecture](./static/architecture.png)

## Prerequisites

To get started, you'll need to install the following tools:

```bash
# Install sqlc for Go code generation from SQL
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Optional: Install go-blueprint for project scaffolding
go install github.com/melkeydev/go-blueprint@latest

# Install golang-migrate for database migrations
brew install golang-migrate
```

### Core Dependencies

The project relies on the following Go modules:

```go
require (
	github.com/gin-contrib/cors v1.7.6
	github.com/gin-gonic/gin v1.10.1
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/sessions v1.4.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.7.5
	github.com/markbates/goth v1.81.0
	github.com/spf13/viper v1.20.1
)
```

## Makefile and Docker

We use a `Makefile` to streamline common development tasks and `Docker` for containerization.

### Makefile

The `Makefile` wraps frequently used commands to speed up development. It dynamically loads environment variables from `.env` for local development and uses production variables when `APP_ENV` is set to `production`.

Key targets include:
- `make build`: Builds the application binary.
- `make run`: Runs the application locally.
- `make test`: Executes all tests.
- `make docker-up`: Starts services using Docker Compose.
- `make migrateup`: Applies all database migrations.
- `make migratedown`: Rolls back all migrations.
- `make sqlc`: Generates Go code from SQL queries.
- `make setup`: A utility command to start Docker containers, wait for the database, and run migrations.
- `make watch`: Enables live reloading with `air`.

**Makefile Snippet:**
```makefile
# Simple Makefile for a Go project
ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
endif

# Container names for Docker operations
DB_CONTAINER_NAME=psql_bp

# Database URL for migrations
DB_URL_LOCAL=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@localhost:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=disable
DB_URL_PROD=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@${BLUEPRINT_DB_HOST}:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=require
DB_URL=$(if $(filter production,$(APP_ENV)),$(DB_URL_PROD),$(DB_URL_LOCAL))

# ... other commands

setup: docker-up wait-for-db migrateup
	@echo "Backend services are ready!"

wait-for-db:
	@echo "Waiting for database to be ready..."
	# ... logic to check if DB is ready
```

### Dockerfile

The `Dockerfile` uses a multi-stage build to create a small, optimized production image.

- **Builder Stage**: Compiles the Go application in a `golang:1.24.4-alpine` image.
- **Production Stage**: Copies the binary and necessary certificates into a lightweight `alpine:3.20.1` image. It also sets up a non-root user for security.
- **Health Check**: A `HEALTHCHECK` is included to ensure the container is running properly.

**Dockerfile Snippet:**
```dockerfile
FROM golang:1.24.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o main cmd/api/main.go

# Production stage
FROM alpine:3.20.1 AS prod

COPY --from=builder /app/main /main
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -fsS "http://localhost:${PORT:-8080}/health" || exit 1
ENTRYPOINT ["/main"]
```

### docker-compose.yml

The `docker-compose.yml` file is used to set up a local PostgreSQL database for development and testing.

```yaml
services:
  psql_bp:
    container_name: psql_bp
    image: postgres:17.5-alpine3.22
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${BLUEPRINT_DB_DATABASE}
      POSTGRES_USER: ${BLUEPRINT_DB_USERNAME}
      POSTGRES_PASSWORD: ${BLUEPRINT_DB_PASSWORD}
    ports:
      - "${BLUEPRINT_DB_PORT}:5432"
    volumes:
      - psql_volume_tomlord:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "sh -c 'pg_isready -U ${BLUEPRINT_DB_USERNAME} -d ${BLUEPRINT_DB_DATABASE}'"]
      # ... healthcheck params
networks:
  tomlord_network:
```

## PostgreSQL DB Schema and Migrations

We use `golang-migrate` for schema migrations and `sqlc` to generate type-safe Go code from SQL.

### sqlc Setup

After running `sqlc init`, configure `sqlc.yaml` to define input directories and output packages.

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "./sqlc/queries"
    schema: "./sqlc/migrations"
    gen:
      go:
        package: "db"
        out: "./internal/db_sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true 
```

- `emit_json_tags`: Serializes struct fields with `json` tags for API responses.
- `emit_empty_slices`: Ensures that queries returning multiple rows result in an empty slice instead of `nil`.
- `emit_interface`: Generates a `Querier` interface for all queries, facilitating mocking and testing.

### Migration Files

Migration files are stored in `./sqlc/migrations`. Each migration has an `up` and a `down` file.

- **`001_create_users_table.up.sql`**: Creates the `users` table to store profile information from Google OAuth.
- **`002_create_messages_table.up.sql`**: Creates the `messages` table with a foreign key to `users`.
- **`003_create_message_thumbs_table.up.sql`**: A linking table for message likes, with a unique constraint on `(message_id, user_id)`. `ON DELETE CASCADE` is used to maintain data integrity.
- **`004_create_blogs_table.up.sql`**: Creates the `blogs` table with support for tags (as a text array) and a GIN index for efficient tag searching.
- **`005_add_foreign_key_post_slug.up.sql`**: Adds a foreign key constraint from `messages.post_slug` to `blogs.slug`.

## SQLC CRUD Code

SQL queries for `sqlc` are located in `./sqlc/queries`. Each query must have a special comment header that `sqlc` uses as an entry point.

- **`-- name: <QueryName> :one`**: Expects the query to return a single row.
- **`-- name: <QueryName> :many`**: Expects the query to return multiple rows.
- **`-- name: <QueryName> :exec`**: Expects the query to perform an action without returning rows.

### Query Files

- **`users.sql`**: Contains queries for creating and retrieving users.
- **`messages.sql`**: Includes queries for CRUD operations on messages, fetching messages by blog slug, and updating thumb counts. It uses `LEFT JOIN` and `COALESCE` to correctly handle messages with no thumbs.
- **`message_thumbs.sql`**: Manages message likes. It uses `ON CONFLICT DO NOTHING` for idempotent like creation and `EXISTS()` to check if a user has already liked a message.
- **`blogs.sql`**: Handles blog post retrieval, including filtering by tag or language. It also contains a query to fetch a blog post along with its message count.

### Generated Go Code

After running `sqlc generate`, the following files are created in `./internal/db_sqlc`:

- **`models.go`**: Contains Go struct definitions for each database table (e.g., `Blog`, `User`). Fields use `pgtype` to handle nullable values correctly.
- **`querier.go`**: Exports a `Querier` interface that lists all generated query methods. The service layer should depend on this interface.
- **`db.go`**: Provides a `New()` constructor to create a `*Queries` struct and a `WithTx()` method for running queries within a transaction.

The service layer (`internal/services/*`) interacts with the database exclusively through the `Querier` interface provided by our `DBService`, which offers transaction support via `WithTx`.

## Build the Server

### Entrypoint: `cmd/api/main.go`

The application bootstrap process is:
1.  `config.Load()`: Loads environment variables.
2.  `server.NewServer()`: Initializes all services and dependencies.
3.  `server.ListenAndServe()`: Starts the HTTP server.
4.  A graceful shutdown mechanism is set up to handle termination signals.

### Server Assembly: `internal/server/server.go`

The `NewServer` function orchestrates dependency injection:
- Initializes `DBService`, `AuthService`, `MessageService`, and `BlogService`.
- Creates an `AuthMiddleware` instance with the `JWT_SECRET`.
- Starts the `WebSocket Hub` in a separate goroutine.
- Configures and returns an `*http.Server` with the Gin router and appropriate timeouts.

### Routes/Handlers: `internal/server/routes.go`

The router is configured with the following:
- **Global Middleware**: `SetupCORS()` for cross-origin requests.
- **Public Routes**: `/`, `/health`, `/debug/jwt`.
- **OAuth Routes**: `/auth/:provider` and `/auth/:provider/callback` for Google login via Goth.
- **API Routes**:
    - `/api/blogs`: For fetching blog posts. Create/update endpoints are available in non-production environments.
    - `/api/messages`: Full CRUD for messages and a toggle endpoint for likes.
    - `/api/sync-blogs`: A one-time endpoint for syncing blog content from the frontend build.
- **WebSocket Route**: `/ws` handles real-time connections.

## Build the Service Layer

### BlogService: `internal/services/blog.go`

- **Responsibilities**: Handles the business logic for blog posts.
- **DTOs**: Defines request/response structs like `CreateBlogRequest` and `BlogInfo` to decouple the API from the database models.
- **Logic**:
    - Converts input types (e.g., string to `pgtype.Date`).
    - Selects the appropriate `sqlc` query based on request parameters (e.g., `tag`, `lang`).
    - Maps database models back to DTOs for the JSON response, converting `pgtype` values to primitives.

### MessageService: `internal/services/message.go`

- **Responsibilities**: Manages message creation, retrieval, and interactions.
- **Logic**:
    - **Create**: After inserting a new message, it re-fetches it to populate user details (`user_name`, `user_picture_url`).
    - **Delete**: Implements two deletion paths: one for the message owner and another for a superuser.
    - **Toggle Like**: Uses a database transaction (`WithTx`) to atomically check if a like exists, then either creates or deletes it. The message's `thumb_count` is updated within the same transaction to prevent race conditions.

## Setup Configuration and CORS

### Configuration: `internal/config/config.go`

- **Viper**: Used to load configuration from environment variables.
- **`.env` Support**: Automatically loads a `.env` file if `APP_ENV` is `local` or not set.
- **Defaults**: Sets default values for `PORT` and `FRONTEND_URL`.

### CORS: `internal/server/cors.go` & `internal/originpolicy/origins.go`

- **Centralized Policy**: `originpolicy.AllowedOrigins()` serves as the single source of truth for allowed origins for both HTTP and WebSocket connections.
- **Environment-Specific**:
    - **Production**: Uses `ALLOWED_ORIGINS` env var, falls back to `FRONTEND_URL`, and finally defaults to `https://tomlord.fyi`.
    - **Development**: Defaults to `http://localhost:5173`.
- **Gin Middleware**: `SetupCORS()` configures the `gin-contrib/cors` middleware with the allowed origins.

## Build Middleware

### JWT Middleware: `internal/middleware/auth.go`

- **`GenerateJWT`**: Signs a 1-hour JWT containing user claims.
- **`ValidateJWT`**: Verifies the token signature and parses claims.
- **`RequireAuth`**: A middleware that aborts the request if a valid JWT is not present. It injects user info into the Gin context.
- **`OptionalAuth`**: A middleware that injects user info if a token is present but does not fail if it's missing. Used for public endpoints that can show extra info to logged-in users.
- **`RequireSuperUserOrOwner`**: Checks for superuser privileges (based on a hardcoded email) or ownership.
- **`RequireSyncToken`**: A separate authentication check for the `/api/sync-blogs` endpoint.

### OAuth and User Upsert: `internal/auth/oauth.go`

- **Goth**: Handles the OAuth2 flow with Google.
- **User Upsert**: In the callback handler, `AuthService.CreateOrUpdateUser` is called. It checks for the user's existence by `google_id` and either creates a new user or updates the existing one.
- **Redirect**: After a successful login, it generates a JWT, sets it in a cookie, and redirects the user to the frontend with the token in the query parameters.

## Build WebSocket

### Core: `internal/websocket/hub.go`

- **`Hub`**: Manages clients, rooms, and message broadcasting.
- **`Client`**: Represents a single WebSocket connection, holding its subscribed rooms and a buffered channel for outgoing messages.
- **Origin Check**: The upgrader's `CheckOrigin` function uses `originpolicy.AllowedOrigins()` to enforce the same CORS policy as HTTP endpoints.
- **Ping/Pong**: Implements a heartbeat mechanism to detect and clean up dead connections.
- **Dynamic Subscriptions**: Clients can send JSON messages to subscribe to or unsubscribe from rooms (e.g., `{"action": "subscribe", "rooms": ["post-slug-1"]}`).

### Server Integration

- **Route**: `GET /ws` is protected by `OptionalAuth()` to associate connections with a `userID` if available.
- **Broadcasting**: Services broadcast events to the hub when relevant actions occur:
    - `MessageTypeNewComment` on new message creation.
    - `MessageTypeThumbUpdate` when a message is liked/unliked.
    - `MessageTypeCommentDelete` when a message is deleted.
- **Room Naming**: Rooms are named after the `post_slug` to deliver events only to clients viewing that specific blog post.

## Deployment

The project is configured for deployment on Fly.io.

### Fly.io Commands

```bash
# Install Fly CLI
brew install flyctl

# Authenticate with Fly
fly auth login

# Initialize the application
fly init

# Set production secrets
fly secrets set \
  PORT=8080 \
  APP_ENV=production \
  BLUEPRINT_DB_HOST=[your_production_db_host] \
  # ... other secrets
  FRONTEND_URL=[your_frontend_url] \
  ALLOWED_ORIGINS=[your_allowed_origins] \
  SYNC_SESSION_SECRET=[your_session_secret]

# List secrets to verify
fly secrets list

# Deploy the application
fly deploy
```