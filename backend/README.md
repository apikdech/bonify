# Receipt Manager - Backend

Go-based REST API server for the Receipt Manager application.

## Features

- **Receipt Management**: CRUD operations with full item/fee breakdown
- **LLM Vision Integration**: OCR receipt parsing via OpenAI/Anthropic/Google
- **Multi-Currency**: Automatic FX rate conversion
- **Budget Tracking**: Monthly limits with alerts
- **Expense Splitting**: Split costs among users
- **Bot Integration**: Telegram and Discord bot handlers
- **JWT Authentication**: Secure token-based auth with refresh tokens
- **Workflow Engine**: Temporal workflows for async processing

## Tech Stack

- **Language**: Go 1.21+
- **Router**: Chi
- **Database**: PostgreSQL 15+
- **Cache**: Redis
- **Workflows**: Temporal
- **Storage**: MinIO/S3-compatible
- **AI**: OpenAI GPT-4, Anthropic Claude, Google Gemini, Ollama

## Quick Start

### Prerequisites

- Go 1.21 or later
- PostgreSQL 15+
- Redis
- MinIO (or S3-compatible storage)
- Temporal (optional, for workflows)

### Installation

```bash
# Clone and enter backend directory
cd backend

# Copy environment file
cp .env.example .env

# Edit .env with your configuration
# At minimum, set DATABASE_URL and JWT_SECRET

# Install dependencies
go mod download

# Run database migrations
go run cmd/api/main.go
# (Migrations run automatically on startup)
```

### Development

```bash
# Run API server
go run cmd/api/main.go

# Run background worker (separate terminal)
go run cmd/worker/main.go

# Run tests
go test ./...

# Build binary
go build -o api cmd/api/main.go
```

### Docker

```bash
# Build image
docker build -t receipt-manager-api .

# Run with docker-compose (recommended)
docker-compose up -d
```

## Project Structure

```
backend/
├── cmd/
│   ├── api/          # HTTP API server entry point
│   └── worker/       # Background worker entry point
├── internal/
│   ├── bot/          # Telegram & Discord bot handlers
│   ├── config/       # Configuration management
│   ├── db/           # Database connection & migrations
│   ├── handler/      # HTTP handlers (REST API)
│   ├── middleware/   # Auth, rate limiting, etc.
│   ├── model/        # Data models
│   ├── repository/   # Database access layer
│   ├── service/      # Business logic layer
│   └── workflow/     # Temporal workflows & activities
└── go.mod
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration (admin only)
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout

### Receipts
- `GET /api/v1/receipts` - List receipts (with filters)
- `POST /api/v1/receipts` - Create receipt
- `GET /api/v1/receipts/:id` - Get receipt details
- `PATCH /api/v1/receipts/:id` - Update receipt
- `DELETE /api/v1/receipts/:id` - Delete receipt
- `PATCH /api/v1/receipts/:id/confirm` - Confirm receipt
- `PATCH /api/v1/receipts/:id/reject` - Reject receipt
- `GET /api/v1/receipts/export` - Export receipts as CSV

### Tags
- `GET /api/v1/tags` - List tags
- `POST /api/v1/tags` - Create tag
- `PATCH /api/v1/tags/:id` - Update tag
- `DELETE /api/v1/tags/:id` - Delete tag

### Budgets
- `GET /api/v1/budgets` - List budgets
- `POST /api/v1/budgets` - Create budget
- `PATCH /api/v1/budgets/:id` - Update budget
- `DELETE /api/v1/budgets/:id` - Delete budget
- `GET /api/v1/budgets/status` - Get budget vs actual

### Splits
- `GET /api/v1/receipts/:id/splits` - Get receipt splits
- `POST /api/v1/receipts/:id/splits` - Create/update splits
- `GET /api/v1/splits/settlements` - Get settlement summary

### Analytics
- `GET /api/v1/analytics/summary` - Dashboard summary
- `GET /api/v1/analytics/insights` - Spending insights
- `GET /api/v1/analytics/monthly` - Monthly trends
- `GET /api/v1/analytics/by-tag` - Spending by tag
- `GET /api/v1/analytics/by-shop` - Spending by shop

### Users
- `GET /api/v1/user` - Get current user
- `PATCH /api/v1/user` - Update user settings

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `JWT_SECRET` | Yes | - | JWT signing secret (generate with `openssl rand -base64 32`) |
| `REDIS_URL` | No | `redis://localhost:6379` | Redis connection string |
| `SERVER_PORT` | No | `8080` | HTTP server port |
| `SERVER_ENV` | No | `development` | Environment (development/staging/production) |
| `LLM_PROVIDER` | No | `openai` | LLM provider (openai/anthropic/google/ollama) |
| `LLM_MODEL` | No | `gpt-4` | Model name |
| `LLM_API_KEY` | No* | - | API key for cloud LLM providers |
| `MINIO_ENDPOINT` | No | `localhost:9000` | MinIO/S3 endpoint |
| `MINIO_ACCESS_KEY` | No | - | MinIO access key |
| `MINIO_SECRET_KEY` | No | - | MinIO secret key |
| `MINIO_BUCKET` | No | `receipts` | Bucket name for images |
| `TEMPORAL_HOST` | No | `localhost:7233` | Temporal server host |
| `TELEGRAM_BOT_TOKEN` | No | - | Telegram bot token |
| `DISCORD_BOT_TOKEN` | No | - | Discord bot token |

*Required for cloud LLM providers

See `.env.example` for the complete list.

## Database Migrations

Migrations are automatically applied on server startup. They are located in:
- `internal/db/migrations/`

Manual migration (if needed):
```bash
# Using migrate CLI
migrate -path internal/db/migrations -database "postgres://..." up
```

## Development Guidelines

### Adding a New Feature

1. **Model**: Define data structures in `internal/model/`
2. **Migration**: Create SQL migration if schema changes needed
3. **Repository**: Add database operations in `internal/repository/`
4. **Service**: Implement business logic in `internal/service/`
5. **Handler**: Add HTTP handlers in `internal/handler/`
6. **Routes**: Register routes in `cmd/api/main.go`

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/service/...
```

## Architecture

### Layer Structure

```
HTTP Request
    ↓
Handler (input validation, HTTP concerns)
    ↓
Service (business logic)
    ↓
Repository (database access)
    ↓
PostgreSQL
```

### Key Design Patterns

- **Repository Pattern**: Data access abstraction
- **Service Layer**: Business logic encapsulation
- **Dependency Injection**: Interface-based dependencies
- **Middleware Chain**: Auth, rate limiting, logging

## License

MIT
