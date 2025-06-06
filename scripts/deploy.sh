#!/bin/bash

# Deployment script for 2048 Game
set -e

echo "🚀 Deploying 2048 Game..."

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

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed or not in PATH"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed or not in PATH"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_warning ".env file not found, copying from .env.example"
    cp .env.example .env
    print_warning "Please edit .env file with your OAuth2 credentials before running the application"
fi

# Build the application first
print_status "Building application..."
./scripts/build.sh

# Stop existing containers
print_status "Stopping existing containers..."
docker-compose -f docker/docker-compose.yml down

# Build and start containers
print_status "Building and starting Docker containers..."
docker-compose -f docker/docker-compose.yml up --build -d

# Wait for services to be ready
print_status "Waiting for services to start..."
sleep 10

# Check if services are running
print_status "Checking service health..."

# Check PostgreSQL
if docker-compose -f docker/docker-compose.yml exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
    print_success "PostgreSQL is ready"
else
    print_warning "PostgreSQL might not be ready yet"
fi

# Check Redis
if docker-compose -f docker/docker-compose.yml exec -T redis redis-cli ping > /dev/null 2>&1; then
    print_success "Redis is ready"
else
    print_warning "Redis might not be ready yet"
fi

# Check backend
if curl -f http://localhost:6060/health > /dev/null 2>&1; then
    print_success "Backend is ready"
else
    print_warning "Backend might not be ready yet"
fi

print_success "Deployment completed!"
print_status "Application is available at: http://localhost:6060"
print_status "To view logs: docker-compose -f docker/docker-compose.yml logs -f"
print_status "To stop: docker-compose -f docker/docker-compose.yml down"

echo ""
echo "🎮 Your 2048 game is now running!"
echo "📝 Don't forget to configure OAuth2 credentials in .env file"
