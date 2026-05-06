# Simple Makefile for a Go project
ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
endif

# Container names for Docker operations
DB_CONTAINER_NAME=psql_bp

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

# Create a new Supabase migration
migration:
	@test -n "$(name)" || (echo "Usage: make migration name=create_table_name" && exit 1)
	supabase migration new $(name)

# Run all pending migrations against the linked Supabase project
migrateup:
	supabase db push

# Run all pending migrations against local Supabase
migrateup1:
	@echo "Supabase CLI does not support pushing exactly one migration. Use 'make migrateup' after creating one migration."

# Reset local Supabase database and replay migrations
migratedown:
	supabase db reset

# Show Supabase migration status
migratedown1:
	@echo "Supabase CLI does not support remote down migrations. Create a forward migration instead."

# Show local/remote migration history
migration-list:
	supabase migration list

# Setup: Start services and run local Supabase migrations
setup: docker-up wait-for-db
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

.PHONY: all build run test watch docker-up docker-run docker-down migration migrateup migratedown migrateup1 migratedown1 migration-list sqlc setup wait-for-db
