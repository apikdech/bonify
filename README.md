# Receipt Manager

A lightweight personal finance tracking system for small groups (family or friends). Users send receipt photos via Telegram or Discord bots, and the system extracts structured data using LLM vision, stores it in PostgreSQL, and presents it in a PWA dashboard for review, editing, and analytics.

## Quick Start

The fastest way to get started is using Docker Compose:

```bash
# Clone the repository
git clone <repository-url>
cd receipt-manager

# Copy environment file and configure
cp .env.example .env
# Edit .env and set at minimum:
# - JWT_SECRET (generate with: openssl rand -base64 32)
# - LLM_API_KEY (your OpenAI/Anthropic key)

# Start all services
docker-compose up -d

# Wait for services to be ready (about 30 seconds)
docker-compose ps

# The API will be available at http://localhost:8080
# MinIO Console at http://localhost:9001 (minioadmin/minioadmin)
# Temporal UI at http://localhost:8088
```

### Running Backend and Frontend Locally

If you prefer to run the application code locally while using Docker for dependencies:

```bash
# 1. Start only the infrastructure services
docker-compose up -d postgres redis minio temporal temporal-ui

# 2. Wait for services to be ready
sleep 10

# 3. Copy and configure backend environment
cp .env.example backend/.env
# Edit backend/.env with local settings

# 4. Run backend
cd backend
go run cmd/api/main.go

# 5. In another terminal, run frontend
cd frontend
npm install
npm run dev
```

## Project Structure

```
receipt-manager/
├── backend/          # Go API server
│   ├── cmd/api/      # HTTP API entry point
│   ├── cmd/worker/   # Background worker entry point
│   └── internal/     # Application code
├── frontend/         # SvelteKit PWA
│   └── src/          # Application source code
├── docker-compose.yml # Docker orchestration
├── .env.example      # Environment template
└── docs/             # Documentation
```

## Features

### Core Features
- **Receipt Capture**: Send receipt photos via Telegram/Discord bots or manual upload
- **OCR with LLM Vision**: Extract shop names, items, prices, fees, and currency automatically
- **Review & Edit**: Review and correct bot-scanned receipts via PWA
- **Analytics**: Monthly trends, category/tag breakdown, per-shop breakdown
- **Multi-Currency**: Automatic currency conversion with exchange rates
- **Multi-User**: Support for 2-10 users with role-based access

### Phase 3 Features (Latest)
- **CSV Export**: Download receipts as CSV for external analysis
- **Budget Limits**: Set monthly spending limits per tag with progress tracking
- **Budget Alerts**: Visual warnings when approaching or exceeding budget limits
- **Notification Settings**: Toggle notifications for parsing, pending reviews, and budget alerts
- **Expense Split**: Split receipts among users with even or custom amounts

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Input Channels                                         │
│   Telegram bot      Discord bot      PWA (manual)       │
└───────────┬─────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────┐
│  Go API (Chi)        │  HTTP API with JWT auth
│  ├─ Receipt Handler  │
│  ├─ Analytics        │
│  ├─ Budget Service   │
│  └─ Split Service    │
└────────┬─────────────┘
         │
    ┌────┴────┬────────┬────────┐
    ▼         ▼        ▼        ▼
┌───────┐ ┌────────┐ ┌────────┐ ┌──────────┐
│Postgre│ │  LLM   │ │  MinIO │ │  Redis   │
│SQL    │ │ Vision │ │  /S3   │ │          │
└───────┘ └────────┘ └────────┘ └──────────┘
         │              ▲
         └──────────────┘
              Temporal
```

## Docker Services

The `docker-compose.yml` includes:

| Service | Port | Description |
|---------|------|-------------|
| postgres | 5432 | PostgreSQL 15 database |
| redis | 6379 | Redis cache & sessions |
| minio | 9000/9001 | S3-compatible object storage |
| temporal | 7233 | Workflow engine (Temporal) |
| temporal-ui | 8088 | Temporal Web UI |

### Useful Docker Commands

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f postgres

# Stop all services
docker-compose down

# Stop and remove volumes (⚠️ deletes data)
docker-compose down -v

# Restart a specific service
docker-compose restart backend

# Check service health
docker-compose ps
```

## Tech Stack

- **Backend**: Go, Chi router, PostgreSQL, Redis, Temporal
- **Frontend**: SvelteKit 5, TypeScript, Tailwind CSS, Chart.js
- **LLM**: OpenAI GPT-4 (configurable: Anthropic, Google, Ollama)
- **Storage**: MinIO/S3 for receipt images
- **Bots**: Telegram Bot API, Discord Bot API

## Documentation

- [Backend README](./backend/README.md) - API setup, endpoints, development guide
- [Frontend README](./frontend/README.md) - PWA setup, routes, development guide
- [spec.md](./spec.md) - Full project specification

## Environment Variables

### Required Variables

Copy `.env.example` to `.env` and configure:

```bash
# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Get LLM API key from:
# - OpenAI: https://platform.openai.com/api-keys
# - Anthropic: https://console.anthropic.com/
LLM_API_KEY=your-api-key
```

See [.env.example](./.env.example) for all configuration options.

## Development

### Backend Development

```bash
cd backend

# Copy and configure environment
cp .env.example .env
# Edit .env with your settings

# Run tests
go test ./...

# Run API server
go run cmd/api/main.go

# Run worker (in another terminal)
go run cmd/worker/main.go
```

See [backend/README.md](./backend/README.md) for detailed development guide.

### Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Start dev server
npm run dev

# Open http://localhost:5173
```

See [frontend/README.md](./frontend/README.md) for detailed development guide.

## Troubleshooting

### Services won't start

```bash
# Check if ports are already in use
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis
lsof -i :9000  # MinIO
lsof -i :8080  # Backend

# Reset everything
docker-compose down -v
docker-compose up -d
```

### Database connection issues

```bash
# Check PostgreSQL is running
docker-compose ps

# Check logs
docker-compose logs postgres

# Connect manually
docker-compose exec postgres psql -U postgres -d receipt_manager
```

### Backend won't start

- Verify `.env` file exists and has required variables
- Check that `JWT_SECRET` is set
- Ensure PostgreSQL is healthy: `docker-compose ps`

## License

MIT

## Contributing

Contributions welcome! Please read the project specification in [spec.md](./spec.md) before contributing.
