# Simple Makefile for a Go project
ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
endif

# Container names for Docker operations
DB_CONTAINER_NAME=psql_bp

# Database URL for migrations (using environment variables)
# For local development (Docker)
DB_URL_LOCAL=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@localhost:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=disable

# For production (Supabase)
DB_URL_PROD=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@${BLUEPRINT_DB_HOST}:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=require

# Use production URL if APP_ENV is production, otherwise use local
DB_URL=$(if $(filter production,$(APP_ENV)),$(DB_URL_PROD),$(DB_URL_LOCAL))

# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main cmd/api/main.go

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Run the application locally
run:
	@go run cmd/api/main.go

# Start all services with Docker Compose
docker-up:
	@if docker compose up --build -d ; then \
		echo "Services started with Docker Compose V2"; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build -d; \
	fi

# Start services and show logs
docker-run:
	@if docker compose up --build 2> /dev/null ; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Stop all services
docker-down:
	@if docker compose down 2> /dev/null ; then \
		echo "Services stopped with Docker Compose V2"; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Run all migrations up
migrateup:
	@echo "Running migrations with DB_URL: $(DB_URL)"
	migrate -path ./sqlc/migrations -database "$(DB_URL)" -verbose up

# Run single migration up
migrateup1:
	@echo "Running single migration with DB_URL: $(DB_URL)"
	migrate -path ./sqlc/migrations -database "$(DB_URL)" -verbose up 1

# Run all migrations down
migratedown:
	@echo "Rolling back migrations with DB_URL: $(DB_URL)"
	migrate -path ./sqlc/migrations -database "$(DB_URL)" -verbose down

# Run single migration down  
migratedown1:
	@echo "Rolling back single migration with DB_URL: $(DB_URL)"
	migrate -path ./sqlc/migrations -database "$(DB_URL)" -verbose down 1

# Setup: Start services and run migrations
setup: docker-up wait-for-db migrateup
	@echo "Backend services are ready!"

# Wait for database to be ready
wait-for-db:
	@echo "Waiting for database to be ready..."
	@echo "Environment: $(APP_ENV)"
	@echo "Database Host: $(if $(filter production,$(APP_ENV)),${BLUEPRINT_DB_HOST},localhost)"
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if [ "$(APP_ENV)" = "production" ]; then \
			echo "Production mode - skipping Docker health check"; \
			break; \
		fi; \
		if docker exec ${DB_CONTAINER_NAME} pg_isready -U ${BLUEPRINT_DB_USERNAME} -d ${BLUEPRINT_DB_DATABASE} >/dev/null 2>&1; then \
			echo "Database is ready!"; \
			break; \
		fi; \
		echo "Waiting for database... ($$i/10)"; \
		sleep 2; \
	done

# Generate sqlc code
sqlc:
	sqlc generate 

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test watch docker-up docker-run docker-down migrateup migratedown migrateup1 migratedown1 sqlc setup wait-for-db