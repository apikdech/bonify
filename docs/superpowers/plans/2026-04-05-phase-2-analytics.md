# Receipt Manager — Phase 2: Analytics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement analytics dashboard with charts, insights, and filtering capabilities.

**Prerequisites:** Phase 1 Core must be complete (receipts, auth, database working)

**Duration:** ~3 weeks

---

## Overview

Phase 2 adds comprehensive spending analytics to the PWA dashboard. All analytics are scoped per-user (each user sees their own data). Features include:

1. **Global date range picker** - Applied to all analytics pages, persisted in URL
2. **Monthly trends** - Bar chart with year-over-year comparison
3. **Category/tag breakdown** - Donut chart showing spend by tag
4. **Per-shop breakdown** - Ranked table of merchants
5. **Spend insights** - Key metrics and fun facts

---

## Backend Tasks

### Task 1: Analytics Service

**Files:**
- Create: `backend/internal/service/analytics.go`
- Create: `backend/internal/repository/analytics.go`
- Create: `backend/internal/handler/analytics.go`

- [ ] **Step 1: Create analytics repository**

Key queries needed:
- Summary stats (total, count, average) for date range
- Monthly aggregation for trends
- Tag-based aggregation
- Shop-based aggregation with visit counts
- Insights (biggest day, most visited, MoM change)

- [ ] **Step 2: Implement analytics service**

Methods:
- `GetSummary(ctx, userID, from, to) (*Summary, error)`
- `GetMonthlyTrends(ctx, userID, months int) ([]MonthData, error)`
- `GetByTag(ctx, userID, from, to) ([]TagSpend, error)`
- `GetByShop(ctx, userID, from, to) ([]ShopSpend, error)`
- `GetInsights(ctx, userID, from, to) (*Insights, error)`

Handle multi-currency conversion using exchange_rates table.

- [ ] **Step 3: Add analytics handlers**

Routes:
- GET `/api/v1/analytics/summary?from=&to=`
- GET `/api/v1/analytics/monthly?months=12`
- GET `/api/v1/analytics/by-tag?from=&to=`
- GET `/api/v1/analytics/by-shop?from=&to=`
- GET `/api/v1/analytics/insights?from=&to=`

All endpoints require JWT authentication.

---

### Task 2: Exchange Rate Sync

**Files:**
- Create: `backend/internal/service/fx.go`
- Modify: `backend/internal/workflow/fx_sync.go`

- [ ] **Step 1: Create FX service**

```go
type FXService struct {
    fxRepo repository.FXRepo
    client *http.Client
}

// FetchRatesFromFrankfurter fetches rates from api.frankfurter.app
func (s *FXService) FetchRatesFromFrankfurter(ctx context.Context, base string) error
```

- [ ] **Step 2: Implement FX sync workflow activity**

Fetch all rates from Frankfurter API and store in exchange_rates table.

- [ ] **Step 3: Schedule daily workflow**

Register FXSyncWorkflow as a Temporal scheduled workflow running at 2 AM daily.

---

## Frontend Tasks

### Task 3: Date Range Picker

**Files:**
- Create: `frontend/src/components/DateRangePicker.svelte`
- Modify: `frontend/src/lib/stores.ts`

- [ ] **Step 1: Create date range store**

```typescript
// Update stores.ts
export const dateRangeStore = writable({
  from: startOfMonth(new Date()),
  to: new Date(),
  preset: 'this_month' // this_week, this_month, last_month, etc.
});

// Subscribe and update URL params
dateRangeStore.subscribe(range => {
  // Update $page.url.searchParams
});
```

- [ ] **Step 2: Create date range picker component**

Features:
- Preset buttons: This Week, This Month, Last Month, This Quarter, This Year, Custom
- Custom date inputs (from/to)
- Sync with URL params for shareable links

- [ ] **Step 3: Add to analytics layout**

Include picker in analytics section layout, apply to all child routes.

---

### Task 4: Analytics API Client

**Files:**
- Modify: `frontend/src/lib/api.ts`

- [ ] **Add analytics methods**

```typescript
async getAnalyticsSummary(params: { from: string; to: string }): Promise<{
  total_spend: number;
  receipt_count: number;
  avg_per_receipt: number;
}>;

async getMonthlyTrends(params: { months: number }): Promise<{
  months: Array<{
    month: string;
    total: number;
    count: number;
  }>;
}>;

async getByTag(params: { from: string; to: string }): Promise<{
  tags: Array<{
    tag_id: string;
    name: string;
    color: string;
    total: number;
    count: number;
    percentage: number;
  }>;
  untagged_total: number;
}>;

async getByShop(params: { from: string; to: string }): Promise<{
  shops: Array<{
    name: string;
    total: number;
    visit_count: number;
    avg_ticket: number;
    last_visit: string;
  }>;
}>;

async getInsights(params: { from: string; to: string }): Promise<{
  biggest_receipt: { id: string; title: string; total: number; date: string };
  most_visited_shop: { name: string; visits: number };
  mom_change: { percentage: number; absolute: number };
  busiest_day: { day: string; total: number };
}>;
```

---

### Task 5: Monthly Trends Page

**Files:**
- Create: `frontend/src/routes/analytics/monthly/+page.svelte`

- [ ] **Step 1: Create bar chart component**

Use Chart.js with svelte-chartjs wrapper:

```svelte
<script>
  import { Bar } from 'svelte-chartjs';
  import { Chart, BarElement, CategoryScale, LinearScale } from 'chart.js';
  Chart.register(BarElement, CategoryScale, LinearScale);
</script>
```

- [ ] **Step 2: Build trends page**

Features:
- Bar chart: month on X-axis, total spend on Y-axis
- Toggle for year-over-year comparison (overlay last year's data)
- Data table below chart
- Click bar to filter receipts to that month

- [ ] **Step 3: Handle empty states**

Show friendly message when no data exists for the selected period.

---

### Task 6: Category/Tag Breakdown Page

**Files:**
- Create: `frontend/src/routes/analytics/by-tag/+page.svelte`

- [ ] **Step 1: Create donut chart component**

```svelte
<script>
  import { Doughnut } from 'svelte-chartjs';
  import { Chart, ArcElement, Tooltip, Legend } from 'chart.js';
  Chart.register(ArcElement, Tooltip, Legend);
</script>
```

- [ ] **Step 2: Build tag breakdown page**

Features:
- Donut chart with color-coded segments
- "Untagged" as a distinct segment (gray)
- Table: tag name, total, count, % of total
- Click tag to filter receipt list
- Legend with tag colors

---

### Task 7: Per-Shop Breakdown Page

**Files:**
- Create: `frontend/src/routes/analytics/by-shop/+page.svelte`

- [ ] **Create shop breakdown page**

Features:
- Ranked table: shop name, total spend, visits, avg ticket, last visit
- Search/filter by shop name
- Sortable columns
- Click shop name to filter receipts
- Pagination if many shops

---

### Task 8: Analytics Home with Insights

**Files:**
- Create: `frontend/src/routes/analytics/+page.svelte`

- [ ] **Build analytics home page**

Features:
- Date range picker at top
- Quick summary cards: Total spend, Receipt count, Daily average
- Insights panel with:
  - Biggest single receipt this period
  - Most visited shop
  - Month-over-month change (with trend indicator)
  - Busiest day of week
- Links to detailed reports
- Recent activity preview

---

### Task 9: Currency Normalization

**Files:**
- Modify: `frontend/src/lib/api.ts` (add currency handling)
- Modify: All analytics components

- [ ] **Add currency display**

- Always show home currency (from user profile)
- Show original currency in tooltips/details
- Handle missing exchange rates gracefully (exclude from totals with warning)

---

## Integration Checklist

- [ ] Analytics endpoints return correct data
- [ ] Exchange rates sync daily
- [ ] Date range picker updates URL and all charts
- [ ] Charts render correctly on desktop and mobile
- [ ] All analytics pages respect the global date range
- [ ] Clicking chart elements filters receipt lists
- [ ] Empty states are handled gracefully
- [ ] Currency conversion works for multi-currency receipts

---

## Spec References

**Analytics API:** See spec.md Section 7 (Analytics endpoints)
**PWA Routes:** See spec.md Section 11 (/analytics/*)
**Features:** See spec.md Section 10 Phase 2 (Analytics)
**FX Rates:** See spec.md Section 6 (exchange_rates table) and Section 13 (FX_REFRESH_CRON)
