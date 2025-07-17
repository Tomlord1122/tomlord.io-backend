# tomlord.io-backend

Go backend API for tomlord.io website with OAuth authentication, PostgreSQL database, and WebSocket support.

## Features

- 🔐 Google OAuth authentication
- 🗄️ PostgreSQL database with SQLC
- 📝 Blog post management
- 💬 Message/comment system
- 🔌 WebSocket real-time communication
- 🐳 Docker containerization
- 🚀 Fly.io deployment ready

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

## 🚀 Deployment

### Fly.io Deployment (Recommended)

Quick deployment to Fly.io:

```bash
# Install Fly CLI
brew install flyctl  # macOS
# curl -L https://fly.io/install.sh | sh  # Linux

# Login to Fly.io
flyctl auth login

# Auto deploy
make fly-auto-deploy
```

For detailed deployment instructions, see:
- [FLY_QUICK_START.md](FLY_QUICK_START.md) - Quick start guide
- [FLY_DEPLOYMENT_GUIDE.md](FLY_DEPLOYMENT_GUIDE.md) - Complete deployment guide

### Fly.io Commands

```bash
# Deploy to Fly.io
make fly-deploy

# Check status
make fly-status

# View logs
make fly-logs

# Open dashboard
make fly-dashboard

# List secrets
make fly-secrets-list

# Create PostgreSQL database
make fly-postgres-create

# Connect to database
make fly-postgres-connect
```

### AWS Deployment (Alternative)

For AWS deployment instructions, see:
- [AWS_QUICK_START.md](AWS_QUICK_START.md) - Quick start guide
- [AWS_DEPLOYMENT_GUIDE.md](AWS_DEPLOYMENT_GUIDE.md) - Complete deployment guide
