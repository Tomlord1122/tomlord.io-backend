# Simple Makefile for a Go project
include .env
export
# Database URL for migrations (using environment variables)
DB_URL=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@localhost:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=disable

# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main cmd/api/main.go

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

# Show service logs
docker-logs:
	@if docker compose logs -f ; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose logs -f; \
	fi

# Create database (if not exists)
createdb:
	@echo "Database should be created automatically by Docker Compose"
	@docker exec -it psql_bp psql -U ${BLUEPRINT_DB_USERNAME} -d ${BLUEPRINT_DB_DATABASE} -c "SELECT 1;" || \
	docker exec -it psql_bp createdb --username=${BLUEPRINT_DB_USERNAME} --owner=${BLUEPRINT_DB_USERNAME} ${BLUEPRINT_DB_DATABASE}

# Drop database
dropdb:
	docker exec -it psql_bp dropdb --username=${BLUEPRINT_DB_USERNAME} ${BLUEPRINT_DB_DATABASE}

# Run all migrations up
migrateup:
	migrate -path migrations -database "$(DB_URL)" -verbose up

# Run single migration up
migrateup1:
	migrate -path migrations -database "$(DB_URL)" -verbose up 1

# Run all migrations down
migratedown:
	migrate -path migrations -database "$(DB_URL)" -verbose down

# Run single migration down  
migratedown1:
	migrate -path migrations -database "$(DB_URL)" -verbose down 1

# Create new migration
new_migration:
	migrate create -ext sql -dir migrations -seq $(name)

# Setup: Start services and run migrations
setup: docker-up wait-for-db migrateup
	@echo "Backend services are ready!"

# Wait for database to be ready
wait-for-db:
	@echo "Waiting for database to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker exec psql_bp pg_isready -U ${BLUEPRINT_DB_USERNAME} -d ${BLUEPRINT_DB_DATABASE} >/dev/null 2>&1; then \
			echo "Database is ready!"; \
			break; \
		fi; \
		echo "Waiting for database... ($$i/10)"; \
		sleep 2; \
	done

# Generate sqlc code
sqlc:
	sqlc generate

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Integration Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

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

.PHONY: all build run test clean watch docker-up docker-run docker-down docker-logs createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc setup wait-for-db itest