# Receipt Manager

A lightweight personal finance tracking system for small groups (family or friends). Users send receipt photos via Telegram or Discord bots, and the system extracts structured data using LLM vision, stores it in PostgreSQL, and presents it in a PWA dashboard for review, editing, and analytics.

## Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd receipt-manager

# Copy environment files
cp .env.example .env

# Start all services
docker-compose up -d

# Or run locally (see backend/README.md and frontend/README.md)
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
    ┌────┴────┬────────┐
    ▼         ▼        ▼
┌───────┐ ┌────────┐ ┌─────────┐
│Postgre│ │  LLM   │ │  MinIO  │
│SQL    │ │ Vision │ │  /S3    │
└───────┘ └────────┘ └─────────┘
```

## Tech Stack

- **Backend**: Go, Chi router, PostgreSQL, Redis, Temporal
- **Frontend**: SvelteKit 5, TypeScript, Tailwind CSS, Chart.js
- **LLM**: OpenAI GPT-4 (configurable)
- **Storage**: MinIO/S3 for receipt images
- **Bots**: Telegram Bot API, Discord Bot API

## Documentation

- [Backend README](./backend/README.md) - API setup and development
- [Frontend README](./frontend/README.md) - PWA setup and development
- [spec.md](./spec.md) - Full project specification

## Environment Variables

See [.env.example](./.env.example) for all configuration options.

## License

MIT
