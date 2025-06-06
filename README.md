# 2048 Game - H5 Implementation

纯 Vibe coding 项目，演示效果：https://2048.linux.do

A complete 2048 game implementation with client-server architecture, OAuth2 authentication, and leaderboards.

## Features

- **Victory Condition**: Game ends when two 8192 tiles merge
- **Mobile Compatible**: Responsive design with touch support
- **Dark Mode**: Optional dark theme toggle
- **Real-time Communication**: WebSocket-based client-server communication
- **Authentication**: OAuth2 integration for user login
- **Leaderboards**: Daily, weekly, monthly, and all-time rankings
- **Production Ready**: Docker deployment with embedded static files

## Tech Stack

### Frontend
- **Runtime**: Bun
- **Languages**: HTML5, CSS3, JavaScript (ES6+)
- **Communication**: WebSocket
- **Build**: Bun bundler

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL
- **Cache**: Redis (optional)
- **Authentication**: OAuth2
- **Communication**: WebSocket (gorilla/websocket)

### Deployment
- **Containerization**: Docker & Docker Compose
- **Static Files**: Embedded in Go binary
- **Database**: PostgreSQL container
- **Cache**: Redis container

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Bun (for development)
- Go 1.21+ (for development)

### Development Setup

1. **Clone and setup**:
```bash
git clone <repository>
cd game2048
cp .env.example .env
# Edit .env with your OAuth2 credentials
```

2. **Start development environment**:
```bash
# Start database
docker-compose up -d postgres redis

# Start backend (development mode)
cd backend
go run cmd/server/main.go

# Start frontend (development mode)
cd frontend
bun install
bun dev
```

### Production Deployment

```bash
# Build and deploy everything
docker-compose up -d
```

The game will be available at `http://localhost:6060`

## Architecture

### Client-Server Communication
- Frontend sends user inputs (swipe/key directions) via WebSocket
- Server processes game logic and returns updated game state
- Server manages scoring, victory conditions, and game persistence
- Real-time leaderboard updates

### Authentication Flow
1. User clicks "Login" → Redirected to OAuth2 provider
2. OAuth2 callback → Server validates and creates session
3. WebSocket connection established with authenticated session
4. Game state tied to user account

### Database Schema
- **users**: User profiles from OAuth2
- **games**: Game sessions and final scores
- **leaderboards**: Cached ranking data
- **daily_scores**, **weekly_scores**, **monthly_scores**: Time-based rankings

## Development

### Frontend Development
```bash
cd frontend
bun install
bun dev        # Development server
bun build      # Production build
```

### Backend Development
```bash
cd backend
go mod tidy
go run cmd/server/main.go    # Development server
go build -o bin/server cmd/server/main.go  # Production build
```

### Database Migrations
```bash
cd backend
go run migrations/migrate.go up    # Apply migrations
go run migrations/migrate.go down  # Rollback migrations
```

## Configuration

Environment variables (see `.env.example`):

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=game2048
DB_USER=postgres
DB_PASSWORD=password

# Redis (optional)
REDIS_HOST=localhost
REDIS_PORT=6379

# OAuth2
OAUTH2_CLIENT_ID=your_client_id
OAUTH2_CLIENT_SECRET=your_client_secret
OAUTH2_REDIRECT_URL=http://localhost:6060/auth/callback
OAUTH2_PROVIDER=google  # google, github, etc.

# Server
SERVER_PORT=6060
JWT_SECRET=your_jwt_secret
```

## API Documentation

### WebSocket Events

**Client → Server**:
- `move`: `{direction: "up|down|left|right"}`
- `new_game`: `{}`
- `get_leaderboard`: `{type: "daily|weekly|monthly|all"}`

**Server → Client**:
- `game_state`: `{board: [[]], score: number, gameOver: boolean, victory: boolean}`
- `leaderboard`: `{rankings: [{user: string, score: number, rank: number}]}`
- `error`: `{message: string}`

## License

MIT License
