# Receipt Manager — Project Specification

> Version 1.1 — April 2026  
> Status: Updated — decisions resolved on questions 1–6

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Goals & Non-Goals](#2-goals--non-goals)
3. [Users & Roles](#3-users--roles)
4. [System Architecture](#4-system-architecture)
5. [Tech Stack](#5-tech-stack)
6. [Data Models & Schema](#6-data-models--schema)
7. [Backend API Specification](#7-backend-api-specification)
8. [Bot Integration](#8-bot-integration)
9. [LLM Vision Pipeline](#9-llm-vision-pipeline)
10. [PWA Dashboard — Features](#10-pwa-dashboard--features)
11. [PWA Dashboard — Routes & Pages](#11-pwa-dashboard--routes--pages)
12. [Non-Functional Requirements](#12-non-functional-requirements)
13. [Infrastructure & Deployment](#13-infrastructure--deployment)
14. [Implementation Phases](#14-implementation-phases)
15. [Open Questions](#15-open-questions)

---

## 1. Project Overview

Receipt Manager is a lightweight personal finance tracking system for a small group (family or friends). Users send receipt photos via Telegram or Discord bots. The backend uses an LLM vision API to extract structured data from the image, stores it in PostgreSQL, and presents it in a PWA dashboard for review, editing, analytics, and manual entry.

### Core loop

```
User sends photo → Bot → Golang API → LLM Vision → Parse JSON → Save to DB → Notify user
User opens PWA → Reviews pending receipts → Explores spending analytics
```

---

## 2. Goals & Non-Goals

### Goals

- Accept receipt images from Telegram and Discord bots
- Extract structured receipt data using LLM vision (shop name, items, prices, fees, currency)
- Store receipts in PostgreSQL with full line-item detail
- Allow users to review, correct, and confirm bot-scanned receipts via a PWA
- Allow manual receipt entry via the PWA
- Display spending analytics: monthly trends, category/tag breakdown, per-shop breakdown
- Support a small multi-user group (2–10 people) with per-user accounts
- Be deployable via Docker Compose on a single VPS

### Non-Goals (for now)

- Native mobile apps (iOS / Android)
- Expense splitting / settlement tracking — deferred to Phase 3
- Public registration / open sign-up — invite-only only
- Real-time collaboration or live sync between users
- Automated bank/card import
- Receipt OCR without an LLM (no Tesseract fallback)

---

## 3. Users & Roles

| Role | Description |
|------|-------------|
| `admin` | Can manage users, view all receipts, access system settings |
| `member` | Can upload receipts, view and edit their own receipts, view group analytics |

All users in the group share a single analytics view (group-level spend). Receipts are owned by the user who submitted them. Admin can view receipts from all users.

### Authentication

- Email + password login (bcrypt hashed)
- JWT access token (15 min TTL) + refresh token (7 days, stored in httpOnly cookie)
- Optional: link a Telegram or Discord user ID to an account for bot identification
- No OAuth / social login in Phase 1

---

## 4. System Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Input channels                                          │
│  Telegram bot      Discord bot      PWA (manual entry)  │
└───────────┬──────────────┬───────────────┬─────────────┘
            │              │               │
            └──────────────▼───────────────┘
                  ┌──────────────────┐
                  │  API Gateway     │  chi router · JWT auth
                  │  (Golang)        │  rate limiting · logging
                  └────────┬─────────┘
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
   ┌──────────────┐ ┌────────────┐ ┌────────────────┐
   │ Receipt      │ │ LLM Vision │ │ Manual Entry   │
   │ Processor    │ │ Service    │ │ Service        │
   └──────┬───────┘ └─────┬──────┘ └───────┬────────┘
          │               │                │
          └───────────────▼────────────────┘
                  ┌──────────────────┐
                  │   PostgreSQL     │
                  │   Redis cache    │
                  │   MinIO / S3     │
                  └──────────────────┘
```

### Key design decisions

- Handler → Service → Repository is the strict call direction. Handlers never touch the DB directly.
- The bot webhook handler and the REST API handler both call the same `service/receipt.go` — no duplicated logic.
- Receipt images are stored in object storage (MinIO or S3). The DB stores only the URL.
- LLM calls are dispatched as Temporal workflows. The API returns `202 Accepted` with a workflow ID immediately; the bot polls or receives a push notification when the workflow completes.
- Object storage is MinIO (self-hosted, S3-compatible). No external cloud storage dependency.

---

## 5. Tech Stack

### Backend

| Concern | Choice | Notes |
|---------|--------|-------|
| Language | Go 1.25+ | Required by any-llm-go |
| Router | `go-chi/chi v5` | Closest to `net/http`, idiomatic |
| Database driver | `jackc/pgx v5` | pgx native pool, no ORM |
| Migrations | `golang-migrate` | SQL files, run on startup |
| Auth | `golang-jwt/jwt v5` | Access + refresh token pattern |
| Object storage | MinIO (self-hosted) | S3-compatible SDK via `minio-go` |
| Cache / sessions | Redis 7 via `redis/go-redis v9` | Rate limiting, refresh tokens |
| Config | `joho/godotenv` | `.env` file + OS env |
| Logging | `log/slog` (stdlib) | Structured JSON logs |
| Async workflows | Temporal (self-hosted) | `go.temporal.io/sdk` — receipt parse, retries, visibility |
| LLM client | `mozilla-ai/any-llm-go` | Unified interface; provider + model configurable via env or admin settings |

### Frontend

| Concern | Choice | Notes |
|---------|--------|-------|
| Framework | SvelteKit | File-based routing, SSR optional |
| Build tool | Vite (bundled with SvelteKit) | |
| PWA | `vite-plugin-pwa` | Manifest, service worker, installable |
| Styling | Tailwind CSS v4 | Utility-first, JIT |
| Charts | Chart.js v4 | Bar, donut, line — lightweight |
| HTTP client | `fetch` with a typed wrapper | No Axios needed |
| State | Svelte stores | Global: auth, date range, current user |
| Adapter | `@sveltejs/adapter-static` | Deploy as SPA from Golang static file server |

### Infrastructure

| Concern | Choice |
|---------|--------|
| Container | Docker + Docker Compose |
| Reverse proxy | Caddy (auto HTTPS) |
| Database | PostgreSQL 16 |
| Image storage | MinIO (self-hosted Docker) |
| Cache | Redis 7 (Docker) |
| Workflow engine | Temporal (self-hosted Docker) |
| CI | GitHub Actions |

---

## 6. Data Models & Schema

### Table: `users`

```sql
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  email         TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  telegram_id   TEXT UNIQUE,
  discord_id    TEXT UNIQUE,
  role          TEXT NOT NULL DEFAULT 'member',  -- member | admin
  -- LLM settings: override system defaults per user (admin can also set globally via settings table)
  llm_provider  TEXT,    -- null = use system default; e.g. anthropic | openai | gemini | ollama
  llm_model     TEXT,    -- null = use system default; e.g. claude-opus-4-5, gpt-4o
  -- Analytics home currency
  home_currency CHAR(3) NOT NULL DEFAULT 'IDR',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Table: `system_settings`

Global key-value settings managed by admin via dashboard. LLM settings here are the system-wide defaults; user-level overrides in `users` take precedence.

```sql
CREATE TABLE system_settings (
  key        TEXT PRIMARY KEY,  -- e.g. llm_provider, llm_model, fx_api_key, ocr_threshold
  value      TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by UUID REFERENCES users(id)
);

-- Seed defaults
INSERT INTO system_settings (key, value) VALUES
  ('llm_provider',       'anthropic'),
  ('llm_model',          'claude-opus-4-5'),
  ('ocr_threshold',      '0.85'),
  ('fx_base_currency',   'IDR'),
  ('fx_provider',        'frankfurter');  -- free open-source exchange rate API
```

### Table: `receipts`

```sql
CREATE TABLE receipts (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title           TEXT,                          -- shop / merchant name
  source          TEXT NOT NULL DEFAULT 'manual', -- telegram | discord | manual
  image_url       TEXT,                          -- object storage path
  ocr_confidence  NUMERIC(3,2),                  -- 0.00–1.00, null if manual
  currency        CHAR(3) NOT NULL DEFAULT 'IDR', -- ISO 4217
  payment_method  TEXT,                          -- cash | card | qris | transfer | unknown
  subtotal        NUMERIC(14,2) NOT NULL DEFAULT 0,
  total           NUMERIC(14,2) NOT NULL DEFAULT 0,
  status          TEXT NOT NULL DEFAULT 'confirmed', -- confirmed | pending_review | rejected
  notes           TEXT,
  receipt_date    DATE,
  paid_by         UUID REFERENCES users(id),     -- for future split feature
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_receipts_user_id    ON receipts(user_id);
CREATE INDEX idx_receipts_date       ON receipts(receipt_date DESC);
CREATE INDEX idx_receipts_status     ON receipts(status);
CREATE INDEX idx_receipts_currency   ON receipts(currency);
```

### Table: `receipt_items`

```sql
CREATE TABLE receipt_items (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  receipt_id  UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  quantity    INT NOT NULL DEFAULT 1,
  unit_price  NUMERIC(14,2) NOT NULL,
  discount    NUMERIC(14,2) NOT NULL DEFAULT 0,
  subtotal    NUMERIC(14,2) NOT NULL
);

CREATE INDEX idx_receipt_items_receipt_id ON receipt_items(receipt_id);
```

### Table: `receipt_fees`

```sql
CREATE TABLE receipt_fees (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  receipt_id  UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
  label       TEXT NOT NULL,        -- e.g. "PPN 11%", "Service charge"
  fee_type    TEXT NOT NULL,        -- tax | service | delivery | tip | other
  amount      NUMERIC(14,2) NOT NULL
);
```

### Table: `exchange_rates`

Cached rates fetched from a live FX API (Frankfurter — free, open-source, ECB-backed). Rates are refreshed daily by a Temporal scheduled workflow. Used to normalize multi-currency analytics to each user's `home_currency`.

```sql
CREATE TABLE exchange_rates (
  base_currency   CHAR(3) NOT NULL,
  target_currency CHAR(3) NOT NULL,
  rate            NUMERIC(18,8) NOT NULL,
  fetched_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (base_currency, target_currency)
);
```

### Table: `tags`

```sql
CREATE TABLE tags (
  id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name    TEXT NOT NULL,
  color   CHAR(7) NOT NULL DEFAULT '#6366f1',  -- hex color for UI
  UNIQUE (user_id, name)
);
```

### Table: `receipt_tags` (join)

```sql
CREATE TABLE receipt_tags (
  receipt_id  UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
  tag_id      UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (receipt_id, tag_id)
);
```

### Table: `budgets` (Phase 3 — create column now, build UI later)

```sql
CREATE TABLE budgets (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tag_id      UUID REFERENCES tags(id) ON DELETE SET NULL,
  month       CHAR(7) NOT NULL,   -- YYYY-MM
  amount_limit NUMERIC(14,2) NOT NULL,
  UNIQUE (user_id, tag_id, month)
);
```

---

## 7. Backend API Specification

Base path: `/api/v1`  
All endpoints except `/auth/*` require `Authorization: Bearer <token>` header.  
All responses are `application/json`.

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/login` | Email + password → access token + refresh token |
| POST | `/auth/refresh` | Refresh token → new access token |
| POST | `/auth/logout` | Invalidate refresh token |
| GET | `/auth/me` | Return current user profile |

### Receipts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/receipts` | List receipts. Query: `status`, `tag`, `from`, `to`, `q`, `page`, `limit` |
| POST | `/receipts` | Create receipt manually (JSON body) |
| GET | `/receipts/:id` | Get single receipt with items, fees, tags |
| PATCH | `/receipts/:id` | Update receipt fields (partial) |
| DELETE | `/receipts/:id` | Soft-delete receipt |
| PATCH | `/receipts/:id/confirm` | Set status to `confirmed` |
| PATCH | `/receipts/:id/reject` | Set status to `rejected` |
| POST | `/receipts/:id/reparse` | Re-run LLM vision on existing image |
| POST | `/receipts/upload` | Upload image → start Temporal workflow → return `202 + workflow_id` |
| GET | `/receipts/jobs/:workflow_id` | Poll Temporal workflow status → `{ status, receipt_id? }` |

### Receipt Items

| Method | Path | Description |
|--------|------|-------------|
| POST | `/receipts/:id/items` | Add item to receipt |
| PATCH | `/receipts/:id/items/:item_id` | Update item |
| DELETE | `/receipts/:id/items/:item_id` | Delete item |

### Tags

| Method | Path | Description |
|--------|------|-------------|
| GET | `/tags` | List user's tags |
| POST | `/tags` | Create tag |
| PATCH | `/tags/:id` | Update tag name / color |
| DELETE | `/tags/:id` | Delete tag |
| POST | `/receipts/:id/tags` | Apply tag to receipt |
| DELETE | `/receipts/:id/tags/:tag_id` | Remove tag from receipt |

### Analytics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/analytics/summary` | Total spend, receipt count, avg per receipt for date range |
| GET | `/analytics/monthly` | Spend per month for last N months |
| GET | `/analytics/by-tag` | Spend grouped by tag for date range |
| GET | `/analytics/by-shop` | Top merchants by total spend, visit count, avg ticket |
| GET | `/analytics/insights` | Biggest day, most visited shop, MoM change |

- All analytics queries are **scoped per user** — each user only sees their own receipts and spend. Admin can optionally view group-wide analytics via a toggle in the admin panel.
- Multi-currency receipts are normalized to the user's `home_currency` using rates from the `exchange_rates` table (fetched daily from Frankfurter API via Temporal scheduled workflow). Raw receipt currency is always shown alongside the converted amount.

### System Settings (admin only)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/settings` | Get all system settings (key-value pairs) |
| PATCH | `/settings` | Update one or more settings |

Settings that can be updated: `llm_provider`, `llm_model`, `ocr_threshold`, `fx_base_currency`, `fx_provider`. Changes take effect immediately — no restart required. The LLM service reads settings on each request (cached in Redis for 60s).

### Users (admin only)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/users` | List all users |
| POST | `/users/invite` | Create a new member account |
| PATCH | `/users/:id` | Update user (name, role) |
| DELETE | `/users/:id` | Deactivate user |

### Bot Webhooks

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhooks/telegram` | Telegram update payload |
| POST | `/webhooks/discord` | Discord interaction / message payload |

---

## 8. Bot Integration

### Telegram

- Use **webhook mode** (not polling) — register URL with `setWebhook`
- Bot secret token verified via `X-Telegram-Bot-Api-Secret-Token` header
- Only messages from known `telegram_id` (linked to a user) are accepted
- On receiving a photo message:
  1. Download the highest-resolution file from Telegram servers
  2. Upload to object storage
  3. Call `service/receipt.ParseReceipt(imageURL)`
  4. If `ocr_confidence >= 0.85` → save as `confirmed`, reply with summary
  5. If `ocr_confidence < 0.85` → save as `pending_review`, reply asking user to review in dashboard
- Supported commands:
  - `/start` — link Telegram account to user (sends a magic link or PIN)
  - `/summary` — reply with this month's total spend
  - `/pending` — reply with count of unreviewed receipts

### Discord

- Use **slash commands + interactions** (not message watching)
- Bot token verified via Ed25519 signature on `X-Signature-Ed25519` header
- Same image processing flow as Telegram
- Slash commands:
  - `/receipt upload` — attach image to parse
  - `/summary` — this month's spend
  - `/pending` — unreviewed count

### Bot reply format (both platforms)

On success:
```
✅ Receipt saved — Kopi Kenangan
📅 5 Apr 2026 · IDR 45,000
🧾 3 items · PPN 11% included
🏷️ No tags yet — add them in the dashboard
```

On pending review:
```
⚠️ Receipt saved but needs your review
The scan confidence is low (62%). Please check the details:
→ https://yourapp.com/receipts/<id>
```

---

## 9. LLM Vision Pipeline

### Overview

Receipt parsing runs as a **Temporal workflow**, not a synchronous HTTP call. When a bot or the PWA uploads an image, the API immediately returns `202 Accepted` with a `workflow_id`. The client polls `GET /receipts/jobs/:workflow_id` for status, or waits for a bot push notification.

```
Image upload
  → API uploads image to MinIO
  → API starts Temporal workflow (ParseReceiptWorkflow)
      → Activity: resolve LLM config (user override → system setting → env fallback)
      → Activity: call LLM via any-llm-go (vision + prompt)
      → Activity: validate + parse JSON response
      → Activity: save receipt + items + fees to PostgreSQL
      → Activity: notify user (bot message or PWA push)
  → API returns 202 + workflow_id
Client polls GET /receipts/jobs/:workflow_id → { status, receipt_id? }
```

### LLM client — `any-llm-go`

`github.com/mozilla-ai/any-llm-go` (v0.8.0, requires Go 1.25+) provides a unified interface across Anthropic, OpenAI, Gemini, Mistral, Ollama, and others. Switching providers requires no code changes — only config.

**Provider resolution order** (first non-empty wins):

1. User-level override (`users.llm_provider` + `users.llm_model`)
2. System setting (`system_settings` table — admin-managed via dashboard)
3. Environment variable fallback (`LLM_PROVIDER` + `LLM_MODEL`)

**Initializing the provider at runtime:**

```go
// internal/service/llm.go

func NewProviderFromConfig(cfg LLMConfig) (anyllm.Provider, error) {
    switch cfg.Provider {
    case "anthropic":
        return anthropic.New(anyllm.WithAPIKey(cfg.APIKey))
    case "openai":
        return openai.New(anyllm.WithAPIKey(cfg.APIKey))
    case "gemini":
        return gemini.New(anyllm.WithAPIKey(cfg.APIKey))
    case "ollama":
        return ollama.New() // no API key needed
    default:
        return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
    }
}
```

**Supported providers and their vision-capable models:**

| Provider | Recommended model | Vision |
|----------|-------------------|--------|
| `anthropic` | `claude-opus-4-5` | ✅ |
| `openai` | `gpt-4o` | ✅ |
| `gemini` | `gemini-1.5-pro` | ✅ |
| `ollama` | `llava` | ✅ (self-hosted) |
| `mistral` | `pixtral-large` | ✅ |

> Note: `any-llm-go` uses `anyllm.CompletionParams` for all providers. Image input is passed via the `Messages` field using the provider's native multimodal content format. Confirm the exact image content block structure against the library's examples for each provider before implementing.

### Temporal workflow definition

```go
// internal/workflow/parse_receipt.go

func ParseReceiptWorkflow(ctx workflow.Context, input ParseReceiptInput) (ParseReceiptResult, error) {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Second,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
            InitialInterval: 2 * time.Second,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var cfg LLMConfig
    if err := workflow.ExecuteActivity(ctx, ResolveLLMConfigActivity, input.UserID).Get(ctx, &cfg); err != nil {
        return ParseReceiptResult{}, err
    }

    var parsed ParsedReceipt
    if err := workflow.ExecuteActivity(ctx, CallLLMVisionActivity, cfg, input.ImageURL).Get(ctx, &parsed); err != nil {
        return ParseReceiptResult{}, err
    }

    var receiptID string
    if err := workflow.ExecuteActivity(ctx, SaveReceiptActivity, input.UserID, parsed).Get(ctx, &receiptID); err != nil {
        return ParseReceiptResult{}, err
    }

    workflow.ExecuteActivity(ctx, NotifyUserActivity, input.UserID, receiptID, parsed.OCRConfidence)

    return ParseReceiptResult{ReceiptID: receiptID}, nil
}
```

Temporal provides built-in retry, timeout, visibility (Temporal UI), and history — no custom retry logic needed in application code.

### System prompt

```
You are a receipt data extraction engine.
Your only job is to extract structured data from receipt images and return
valid JSON. Never add commentary, explanations, or markdown.
If a field cannot be determined from the image, use null.
All monetary values must be numbers (not strings), in the receipt's original currency unit.
```

### User prompt

```
Extract all data from this receipt image and return ONLY a JSON object
with this exact structure:

{
  "title": "string — shop or merchant name",
  "receipt_date": "YYYY-MM-DD or null",
  "currency": "ISO 4217 code e.g. IDR, USD, SGD",
  "payment_method": "cash | card | qris | transfer | unknown",
  "items": [
    {
      "name": "string",
      "quantity": number,
      "unit_price": number,
      "discount": number or 0,
      "subtotal": number
    }
  ],
  "fees": [
    {
      "label": "string e.g. PPN 11%, Service charge, Delivery",
      "fee_type": "tax | service | delivery | tip | other",
      "amount": number
    }
  ],
  "subtotal": number,
  "total": number,
  "ocr_confidence": number between 0.0 and 1.0
}

Rules:
- ocr_confidence reflects how clearly the receipt is readable (1.0 = perfectly clear)
- subtotal is the sum of items before fees
- total is the final amount after all fees and discounts
- If an item has no discount, set discount to 0
- Quantity must be a positive integer
- Do not invent data — use null for genuinely missing fields
```

### Confidence thresholds

Default threshold is `0.85`, configurable via `system_settings.ocr_threshold`.

| Range | Action |
|-------|--------|
| threshold – 1.00 | Auto-confirm, save as `confirmed` |
| 0.60 – (threshold-0.01) | Save as `pending_review`, notify user to review |
| 0.00 – 0.59 | Save as `pending_review`, warn user scan is low quality |

### Error handling

- LLM returns non-JSON → Temporal retries up to 3×, then saves raw response + flags `pending_review`
- LLM API timeout (>30s per activity) → Temporal retries with backoff
- Validation fails → save what was parsed, flag affected fields as unverified
- All failures visible in Temporal UI with full activity history

---

## 10. PWA Dashboard — Features

### Phase 1 — Core (build first)

#### Home overview
- This month's total spend (large, prominent)
- Count of receipts pending review with a link to the queue
- Last 5 receipts as a quick feed (shop name, date, total, status badge)
- Quick action button: "Add receipt" (opens manual entry form)
- Bot connection status (Telegram / Discord linked or not)

#### Review queue
- List of all receipts with `status = pending_review`, newest first
- Each card shows: image thumbnail, OCR confidence badge, shop name (editable), total, date
- Inline editing: tap any field to correct it — shop name, date, total, line items
- Confirm button → sets `status = confirmed`
- Reject button → sets `status = rejected` (stays in history)
- Re-run OCR button → sends image through LLM again
- Empty state: "You're all caught up" when queue is empty

#### Receipt list
- Paginated list of all receipts (20 per page)
- Search by shop name, notes, or item name (full-text)
- Filter by: date range, tag, status, payment method, source (bot vs manual)
- Sort by: date (default), total amount
- Each row: shop name, date, total, currency, tags, source icon, status badge
- Tap row → opens receipt detail page

#### Receipt detail page
- Full receipt view: image (if available), all fields, all items, all fees
- Edit mode: toggle to edit any field — uses same form as manual entry
- Tag selector: add/remove tags with a dropdown
- Delete button (with confirmation)
- "Parsed by AI" badge if source is bot, with confidence score
- Re-run OCR button

#### Manual entry form
- Fields: shop name, date (date picker, default today), currency (default IDR), payment method
- Line items: add/remove rows — name, quantity, unit price, discount
- Auto-calculated subtotals per item and grand subtotal
- Fee rows: add/remove — label, type, amount
- Auto-calculated total
- Notes (optional)
- Tags (optional)
- Save / Cancel buttons

#### Tag manager
- List of user's tags with color swatches
- Create new tag: name + color picker
- Rename tag inline
- Delete tag (with warning: "This will remove the tag from N receipts")
- Bulk apply: select multiple receipts from the list → apply tag

### Phase 2 — Analytics

#### Date range picker (global)
- Stored in a Svelte store, applied to all analytics pages
- Presets: This week, This month, Last month, This quarter, This year, Custom
- Custom: from/to date picker
- Persisted in URL query params so analytics pages are shareable/bookmarkable

#### Monthly trends page
- Bar chart: total spend per month for the selected period
- Overlay toggle: compare to same period last year
- Table below chart: month, receipt count, total, avg per receipt
- Click a bar → filter receipt list to that month

#### Category / tag breakdown page
- Donut chart: spend share per tag
- Table: tag name, total spend, receipt count, % of total
- Click a tag → filter receipt list to that tag
- "Untagged" shown as its own segment

#### Per-shop breakdown page
- Ranked table: shop name, total spend, visit count, average ticket size, last visit date
- Search/filter the table by shop name
- Click a shop → filter receipt list to that shop

#### Spend insights panel (shown on analytics home)
- Biggest single receipt this period
- Most visited shop this period
- Month-over-month change in total spend (% and absolute)
- Day of week you spend most on

### Phase 3 — Polish & Extras

#### Export
- Download all receipts (filtered by current date range and tags) as CSV
- CSV includes: date, shop, items (flattened), total, currency, tags, source
- PDF export (single receipt or batch) — nice-to-have

#### Budget limits
- Set a monthly spend limit per tag
- Progress bar on tag breakdown page showing % of budget used
- Visual warning when >80% of budget is reached
- Notification settings: alert via bot when budget is exceeded

#### Multi-currency view
- Each user sets a `home_currency` in their profile settings (default IDR)
- All analytics totals are normalized to the user's home currency using live rates from the `exchange_rates` table
- Rates are refreshed daily via a Temporal scheduled workflow hitting the Frankfurter API (`api.frankfurter.app`)
- Per-receipt pages always show the original currency; analytics pages show both original and converted
- If a rate is unavailable (rare), the receipt is excluded from normalized totals with a visible warning

#### Expense split (deferred)
- `paid_by` field already exists on `receipts`
- UI: mark which user paid for a receipt, split evenly or by items
- Settlement summary: who owes whom across the group

#### Notification settings
- Toggle: receive Telegram/Discord message when receipt is parsed
- Toggle: receive reminder if review queue has items older than 24h
- Toggle: receive budget warning alerts

---

## 11. PWA Dashboard — Routes & Pages

SvelteKit file-based routing under `src/routes/`:

```
/                          → Home overview
/receipts                  → Receipt list (with search + filters)
/receipts/new              → Manual entry form
/receipts/[id]             → Receipt detail + edit
/queue                     → Review queue (pending_review receipts)
/analytics                 → Analytics home + insights panel
/analytics/monthly         → Monthly trends chart
/analytics/by-tag          → Category breakdown
/analytics/by-shop         → Per-shop breakdown
/tags                      → Tag manager
/settings                  → User settings (name, password, linked bots, home currency, LLM override)
/settings/users            → User management — create accounts, set roles (admin only)
/settings/system           → System settings — LLM provider/model, OCR threshold, FX config (admin only)
/settings/system/temporal  → Temporal workflow monitor link / status (admin only)
/login                     → Login page (unauthenticated)
```

### Global layout

- Left sidebar (desktop) / bottom nav (mobile) with: Home, Receipts, Queue (badge count), Analytics, Tags
- Top bar: date range picker (analytics pages only), user avatar + logout
- All routes under `/` are protected — redirect to `/login` if no valid token

---

## 12. Non-Functional Requirements

### Performance

- Receipt list page must load in < 1s for up to 10,000 receipts (paginated, indexed queries)
- LLM parse workflow must complete within 30s end-to-end; bot notifies user asynchronously on completion
- PWA must achieve Lighthouse PWA score ≥ 90
- Analytics queries must complete in < 500ms for up to 5 years of data

### Security

- All API endpoints require JWT authentication except `/auth/login` and `/auth/refresh`
- Bot webhooks validated by platform-specific signature verification
- Passwords hashed with bcrypt, cost factor 12
- Refresh tokens stored in Redis with TTL, invalidated on logout
- Image uploads: validate MIME type (JPEG, PNG, WEBP only), max 10MB
- Rate limiting: 60 requests/min per user on API, 10 bot image uploads/hour per user
- CORS: allow only the PWA origin

### Reliability

- Go 1.25+ required (minimum version for `any-llm-go`)
- Database migrations run automatically on server startup
- All LLM activities run inside Temporal workflows with a 30s per-activity timeout and up to 3 automatic retries with exponential backoff — no custom retry logic in application code
- Object storage uploads have a 30s timeout
- Structured JSON logs for all requests (method, path, status, latency, user_id)

### Accessibility

- PWA meets WCAG 2.1 AA for all Phase 1 pages
- All form inputs have proper labels
- Color is never the only indicator of meaning (use icons + text alongside color)

---

## 13. Infrastructure & Deployment

### Docker Compose services

```yaml
services:
  api:             # Golang binary, port 8080
  worker:          # Temporal worker process (same binary, different entrypoint flag)
  postgres:        # PostgreSQL 16, volume-mounted data (also used by Temporal)
  redis:           # Redis 7, volume-mounted data
  minio:           # MinIO, ports 9000 (API) + 9001 (console)
  temporal:        # Temporal server, port 7233
  temporal-ui:     # Temporal Web UI, port 8088
  caddy:           # Reverse proxy, auto HTTPS, serves PWA static files
```

> The `worker` and `api` services share the same compiled binary — differentiated by a `--mode=worker` flag. This keeps the Docker image count minimal.

### Environment variables (backend)

```env
# Server
PORT=8080
ENV=production

# Database
DATABASE_URL=postgres://user:pass@postgres:5432/receiptdb

# Redis
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=<random 64-char string>
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# Object storage (MinIO)
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=...
MINIO_SECRET_KEY=...
MINIO_BUCKET=receipts
MINIO_USE_SSL=false

# LLM — system-wide defaults (overridable via admin settings UI or per-user)
# Provider options: anthropic | openai | gemini | mistral | ollama
LLM_PROVIDER=anthropic
LLM_MODEL=claude-opus-4-5
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...        # needed if LLM_PROVIDER=openai
GEMINI_API_KEY=...           # needed if LLM_PROVIDER=gemini
MISTRAL_API_KEY=...          # needed if LLM_PROVIDER=mistral
# OLLAMA_BASE_URL=http://ollama:11434  # needed if LLM_PROVIDER=ollama

# Temporal
TEMPORAL_HOST=temporal:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=receipt-parse

# Exchange rates (Frankfurter — free, no API key needed)
FX_PROVIDER=frankfurter      # https://api.frankfurter.app
FX_BASE_CURRENCY=IDR
FX_REFRESH_CRON=0 2 * * *   # daily at 2am

# Bots
TELEGRAM_BOT_TOKEN=...
TELEGRAM_WEBHOOK_SECRET=...
DISCORD_BOT_TOKEN=...
DISCORD_PUBLIC_KEY=...

# App
APP_URL=https://yourapp.com
OCR_CONFIDENCE_THRESHOLD=0.85  # system default; overridable in admin settings
```

### CI/CD (GitHub Actions)

- On push to `main`: run `go test ./...`, build Docker image, push to registry
- On tag `v*`: deploy to VPS via SSH + `docker compose pull && docker compose up -d`
- Frontend: `pnpm build` → copy `build/` into the Golang binary's static file server

---

## 14. Implementation Phases

### Phase 1 — Core (target: ~6 weeks)

**Week 1–2: Backend foundation**
- Go project scaffold: chi router, middleware, config, slog
- PostgreSQL migrations (all tables including `system_settings`, `exchange_rates`)
- Auth endpoints (login, refresh, logout, me)
- Admin: create user account directly (no invite email — admin sets credentials, user changes password on first login)
- Receipt CRUD endpoints
- Tag CRUD endpoints
- System settings endpoints (admin only)

**Week 2–3: LLM pipeline + bots**
- MinIO integration (upload, presign)
- `any-llm-go` integration — provider factory, config resolution (user → system → env)
- Temporal worker setup — `ParseReceiptWorkflow` with 4 activities
- Temporal scheduled workflow — daily FX rate fetch from Frankfurter
- Telegram bot webhook
- Discord bot webhook
- Bot ↔ user linking

**Week 3–4: Analytics API**
- Summary, monthly, by-tag, by-shop, insights endpoints
- Query optimization + indexes

**Week 4–6: PWA dashboard**
- SvelteKit scaffold + PWA config + Tailwind
- Login page + auth store
- Home overview page
- Receipt list + search + filters
- Receipt detail + edit
- Manual entry form
- Review queue
- Tag manager

### Phase 2 — Analytics (target: ~3 weeks)

- Global date range picker (Svelte store + URL params)
- Monthly trends chart (Chart.js)
- Category/tag breakdown (donut chart)
- Per-shop breakdown (ranked table)
- Spend insights panel

### Phase 3 — Polish (ongoing)

- CSV export
- Budget limits
- Multi-currency normalization
- Notification settings
- Expense split UI (schema already in place)
- PDF export

---

## 15. Resolved Decisions & New Open Questions

### Resolved (v1.1)

| # | Question | Decision |
|---|----------|----------|
| 1 | Self-host LLM vs cloud API? | Cloud API via `any-llm-go` — provider + model configurable via admin settings or env. Ollama remains an option for self-hosted by setting `LLM_PROVIDER=ollama`. |
| 2 | MinIO vs AWS S3? | MinIO self-hosted. S3-compatible so migration to AWS S3 later requires only env var changes. |
| 3 | Async job queue? | Temporal (self-hosted). Replaces the need for `riverqueue/river`. Gives workflow visibility, retries, scheduling, and history out of the box. |
| 4 | Analytics scope — shared or per-user? | Per-user by default. Admin gets an optional group-wide view toggle in the admin panel. |
| 5 | Multi-currency — manual rate or live API? | Live API via Frankfurter (free, ECB-backed, no API key). Rates refreshed daily by a Temporal scheduled workflow and cached in `exchange_rates` table. |
| 6 | Invite flow? | Admin creates accounts directly. No email invite. User receives credentials and must change password on first login. |

### New open questions

| # | Question | Impact | Decision needed by |
|---|----------|--------|-------------------|
| 1 | Temporal storage backend: use the existing PostgreSQL instance or run Temporal with its own dedicated Postgres? | Ops complexity vs resource isolation | Before infrastructure setup |
| 2 | Should the Temporal UI be exposed publicly (behind auth) or only accessible via SSH tunnel / VPN? | Security surface | Before deployment |
| 3 | `any-llm-go` image input format varies per provider — confirm multimodal content block structure for each target provider before implementing the vision activity | Correctness of LLM calls | Before LLM activity implementation |
| 4 | Should per-user LLM overrides be available to all members, or admin only? | UX complexity vs flexibility | Before settings UI build |
