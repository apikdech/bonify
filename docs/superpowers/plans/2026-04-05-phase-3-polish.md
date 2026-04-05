# Receipt Manager — Phase 3: Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add export functionality, budget limits, notification settings, and expense splitting.

**Prerequisites:** Phase 1 and Phase 2 complete

**Duration:** Ongoing (2-4 weeks per feature)

---

## Overview

Phase 3 adds polish features that enhance the core experience:

1. **CSV Export** - Download receipts as CSV
2. **Budget Limits** - Set monthly spending limits per tag
3. **Notification Settings** - Bot notification preferences
4. **Expense Split** - Track who paid and split costs
5. **PDF Export** - Generate PDF receipts (nice-to-have)

---

## Task 1: CSV Export

### Backend

**Files:**
- Modify: `backend/internal/handler/receipt.go`

- [ ] **Add CSV export endpoint**

```go
// GET /api/v1/receipts/export?from=&to=&format=csv
func (h *ReceiptHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
    // Get filtered receipts
    // Generate CSV with headers:
    // date, shop, items (flattened), total, currency, tags, source, status
    // Set Content-Type: text/csv
    // Set Content-Disposition: attachment; filename="receipts_YYYY-MM-DD.csv"
}
```

### Frontend

**Files:**
- Modify: `frontend/src/routes/receipts/+page.svelte`
- Modify: `frontend/src/routes/analytics/+page.svelte`

- [ ] **Add export button**

- Place in receipt list page (exports filtered results)
- Place in analytics home (exports date range)
- Use `<a download>` or `fetch` + Blob for download

---

## Task 2: Budget Limits

### Backend

**Files:**
- Create: `backend/internal/service/budget.go`
- Create: `backend/internal/repository/budget.go`
- Create: `backend/internal/handler/budget.go`

- [ ] **Create budget repository**

```go
// Budgets table already exists from Phase 1
type BudgetRepo struct {
    db *pgxpool.Pool
}

func (r *BudgetRepo) Create(ctx context.Context, budget *model.Budget) error
func (r *BudgetRepo) GetByUserAndMonth(ctx context.Context, userID, month string) ([]model.Budget, error)
func (r *BudgetRepo) GetSpentByTag(ctx context.Context, userID, tagID, month string) (float64, error)
func (r *BudgetRepo) Update(ctx context.Context, budget *model.Budget) error
func (r *BudgetRepo) Delete(ctx context.Context, id string, userID string) error
```

- [ ] **Create budget service**

```go
type BudgetService struct {
    budgetRepo *repository.BudgetRepo
}

// GetBudgetStatus returns budget vs actual for a month
func (s *BudgetService) GetBudgetStatus(ctx context.Context, userID, month string) ([]BudgetStatus, error)

// CheckBudgetAlerts returns alerts for budgets over threshold
func (s *BudgetService) CheckBudgetAlerts(ctx context.Context, userID string) ([]BudgetAlert, error)
```

- [ ] **Add budget handlers**

Routes:
- GET `/api/v1/budgets` - List user's budgets
- POST `/api/v1/budgets` - Create budget
- PATCH `/api/v1/budgets/:id` - Update budget
- DELETE `/api/v1/budgets/:id` - Delete budget
- GET `/api/v1/budgets/status?month=` - Get budget vs actual

### Frontend

**Files:**
- Create: `frontend/src/routes/settings/budgets/+page.svelte`
- Modify: `frontend/src/routes/analytics/by-tag/+page.svelte`

- [ ] **Create budget management page**

Features:
- List existing budgets by month
- Create budget: select tag, set month, set amount
- Edit/delete budgets
- View budget vs actual for selected month

- [ ] **Add budget progress to tag breakdown**

- Progress bar on each tag row
- Color coding: green (<80%), yellow (80-100%), red (>100%)
- Tooltip showing budget amount

- [ ] **Budget alerts**

- Show warning banner when approaching limit
- Alert in notification center

---

## Task 3: Notification Settings

### Backend

**Files:**
- Modify: `backend/internal/model/user.go`
- Modify: `backend/internal/db/migrations/` (add columns)
- Modify: `backend/internal/service/notification.go`

- [ ] **Add notification settings to user model**

```go
type User struct {
    // ... existing fields ...
    NotifyOnParse         bool `json:"notify_on_parse"`
    NotifyOnPendingReview bool `json:"notify_on_pending_review"`
    NotifyBudgetAlerts    bool `json:"notify_budget_alerts"`
}
```

- [ ] **Create migration for new columns**

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_on_parse BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_on_pending_review BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_budget_alerts BOOLEAN DEFAULT true;
```

- [ ] **Create notification service**

```go
type NotificationService struct {
    userRepo *repository.UserRepo
    botNotifier BotNotifier
}

func (s *NotificationService) ShouldNotify(userID string, notificationType string) bool
func (s *NotificationService) SendBudgetAlert(ctx context.Context, userID string, budget *BudgetStatus) error
func (s *NotificationService) SendPendingReminder(ctx context.Context, userID string, count int) error
```

- [ ] **Add scheduled reminder workflow**

Temporal scheduled workflow that runs daily:
- Check for pending receipts older than 24h
- Send reminder if enabled

### Frontend

**Files:**
- Modify: `frontend/src/routes/settings/+page.svelte`

- [ ] **Add notification settings section**

Toggle switches for:
- "Notify when receipt is parsed"
- "Notify when review is needed (24h+)"
- "Notify when approaching budget limit"

---

## Task 4: Expense Split (Advanced)

### Backend

**Files:**
- The `paid_by` field already exists on receipts table
- Create: `backend/internal/model/split.go`
- Create: `backend/internal/repository/split.go`

- [ ] **Create split models and repository**

```go
// receipt_splits table (needs migration)
type ReceiptSplit struct {
    ID        string  `json:"id"`
    ReceiptID string  `json:"receipt_id"`
    UserID    string  `json:"user_id"`
    Amount    float64 `json:"amount"`
    Percentage float64 `json:"percentage"` // For percentage-based splits
}

type SplitRepo struct {
    db *pgxpool.Pool
}

func (r *SplitRepo) CreateSplit(ctx context.Context, split *ReceiptSplit) error
func (r *SplitRepo) GetSplitsByReceipt(ctx context.Context, receiptID string) ([]ReceiptSplit, error)
func (r *SplitRepo) GetSettlementSummary(ctx context.Context, groupID string) ([]Settlement, error)
```

- [ ] **Add split handlers**

Routes:
- POST `/api/v1/receipts/:id/splits` - Define splits for a receipt
- GET `/api/v1/receipts/:id/splits` - Get splits for a receipt
- GET `/api/v1/splits/settlements` - Get who owes whom

### Frontend

**Files:**
- Modify: `frontend/src/routes/receipts/[id]/+page.svelte`
- Create: `frontend/src/components/SplitEditor.svelte`

- [ ] **Add split editor to receipt detail**

- Toggle: "Split this receipt"
- Options: Even split, Custom amounts, By items
- Show split summary
- Settlement view across group

- [ ] **Create settlements page**

- Summary of who owes whom
- Suggest optimal payments to settle
- Mark settlements as paid

---

## Task 5: PDF Export (Nice-to-Have)

### Backend

**Files:**
- Create: `backend/internal/service/pdf.go`

- [ ] **Add PDF generation service**

Use a PDF library (e.g., gofpdf, unidoc):
```go
func (s *PDFService) GenerateReceiptPDF(receipt *model.Receipt) ([]byte, error)
```

- [ ] **Add PDF export endpoint**

```go
// GET /api/v1/receipts/:id/export?format=pdf
func (h *ReceiptHandler) ExportPDF(w http.ResponseWriter, r *http.Request)
```

### Frontend

**Files:**
- Modify: `frontend/src/routes/receipts/[id]/+page.svelte`

- [ ] **Add PDF export button**

Single receipt export from detail page.

---

## Multi-Currency Enhancement

### Frontend

- [ ] **Enhance currency display**

On receipt detail:
- Show original amount and currency prominently
- Show converted amount in home currency
- Show exchange rate used

On analytics:
- Tooltip showing original amounts
- Warning when rates are stale (>24h)

---

## Testing Checklist

### CSV Export
- [ ] Exports filtered receipts correctly
- [ ] CSV format is valid and readable
- [ ] Handles large exports efficiently

### Budget
- [ ] Budgets are created and stored correctly
- [ ] Budget vs actual calculates correctly
- [ ] Alerts trigger at correct thresholds
- [ ] Progress bars update in real-time

### Notifications
- [ ] Settings are saved and loaded correctly
- [ ] Notifications respect user preferences
- [ ] Reminders trigger at correct times

### Expense Split
- [ ] Splits calculate correctly (even and custom)
- [ ] Settlement summary is accurate
- [ ] Paid_by field is updated correctly

---

## Spec References

**Budgets:** See spec.md Section 6 (budgets table) and Section 10 Phase 3
**Notifications:** See spec.md Section 10 Phase 3 (Notification settings)
**Expense Split:** See spec.md Section 10 Phase 3 (Expense split)
**CSV Export:** See spec.md Section 10 Phase 3 (Export)
**PDF Export:** See spec.md Section 10 Phase 3 (nice-to-have)
