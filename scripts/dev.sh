#!/bin/bash

# Development script for 2048 Game
set -e

echo "🎮 Starting 2048 Game in Development Mode..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "backend/go.mod" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_warning ".env file not found, copying from .env.example"
    cp .env.example .env
    print_warning "Please edit .env file with your OAuth2 credentials"
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

# Start database services
print_status "Starting database services..."
docker-compose -f docker/docker-compose.yml up -d postgres redis

# Wait for databases to be ready
print_status "Waiting for databases to be ready..."
sleep 5

# Check database health
print_status "Checking database health..."
for i in {1..30}; do
    if docker-compose -f docker/docker-compose.yml exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
        print_success "PostgreSQL is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "PostgreSQL failed to start"
        exit 1
    fi
    sleep 1
done

for i in {1..30}; do
    if docker-compose -f docker/docker-compose.yml exec -T redis redis-cli ping > /dev/null 2>&1; then
        print_success "Redis is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "Redis failed to start"
        exit 1
    fi
    sleep 1
done

# Build the application
print_status "Building application..."
cd backend
go mod tidy
go build -o bin/server-dev cmd/server/main.go

if [ $? -eq 0 ]; then
    print_success "Application built successfully"
else
    print_error "Failed to build application"
    exit 1
fi

# Start the server
print_status "Starting server..."
print_status "Server will be available at: http://localhost:6060"
print_status "Press Ctrl+C to stop the server"

# Set development environment
export GIN_MODE=debug
export STATIC_FILES_EMBEDDED=false

# Run the server
./bin/server-dev
