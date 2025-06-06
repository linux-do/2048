#!/bin/bash

# Build script for 2048 Game
set -e

echo "🎮 Building 2048 Game..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Create necessary directories
print_status "Creating build directories..."
mkdir -p backend/cmd/server/static/{css,js}
mkdir -p backend/bin

# Copy static files to the backend static directory
print_status "Copying static files..."
cp -r backend/cmd/server/static/* backend/static/ 2>/dev/null || true

# Build the Go backend
print_status "Building Go backend..."
cd backend

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3)
print_status "Using Go version: $GO_VERSION"

# Download dependencies
print_status "Downloading Go dependencies..."
go mod download

# Run tests if they exist
if ls *_test.go 1> /dev/null 2>&1; then
    print_status "Running Go tests..."
    go test ./... -v
else
    print_warning "No Go tests found"
fi

# Build the binary
print_status "Compiling backend binary..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/server cmd/server/main.go

if [ $? -eq 0 ]; then
    print_success "Backend binary built successfully: backend/bin/server"
else
    print_error "Failed to build backend binary"
    exit 1
fi

# Build for current OS as well (for development)
print_status "Building development binary..."
go build -o bin/server-dev cmd/server/main.go

cd ..

# Frontend build (if using Bun)
if command -v bun &> /dev/null; then
    print_status "Building frontend with Bun..."
    cd frontend
    
    if [ -f "package.json" ]; then
        print_status "Installing frontend dependencies..."
        bun install
        
        print_status "Building frontend..."
        bun run build
        
        # Copy built files to backend static directory
        if [ -d "dist" ]; then
            print_status "Copying frontend build to backend..."
            cp -r dist/* ../backend/static/
        fi
    else
        print_warning "No package.json found in frontend directory"
    fi
    
    cd ..
else
    print_warning "Bun not found, skipping frontend build"
fi

# Create deployment package
print_status "Creating deployment package..."
mkdir -p dist
tar -czf dist/game2048-$(date +%Y%m%d-%H%M%S).tar.gz \
    backend/bin/server \
    backend/static \
    docker/docker-compose.yml \
    docker/Dockerfile.backend \
    .env.example \
    README.md

print_success "Build completed successfully!"
print_status "Deployment package created in dist/ directory"
print_status "To run the development server: cd backend && ./bin/server-dev"
print_status "To deploy with Docker: docker-compose up -d"

echo ""
echo "🚀 Ready to launch your 2048 game!"
