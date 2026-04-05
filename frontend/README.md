# Receipt Manager - Frontend

Progressive Web App (PWA) built with SvelteKit for the Receipt Manager application.

## Features

- **Progressive Web App**: Works offline, installable on mobile/desktop
- **Receipt Management**: List, filter, create, edit receipts
- **Receipt Review**: Confirm or reject OCR-scanned receipts
- **Analytics Dashboard**: Charts and insights for spending patterns
- **Budget Tracking**: Set limits, view progress, receive alerts
- **Split Expenses**: Divide costs among users
- **Tag Management**: Organize receipts with color-coded tags
- **Settings**: Notification preferences and user settings
- **CSV Export**: Download receipt data for external analysis

## Tech Stack

- **Framework**: SvelteKit 5 (Svelte 5 runes)
- **Language**: TypeScript
- **Styling**: Tailwind CSS 4
- **Charts**: Chart.js with svelte-chartjs
- **Icons**: Lucide Svelte
- **Build Tool**: Vite
- **PWA**: vite-plugin-pwa

## Quick Start

### Prerequisites

- Node.js 18+ 
- npm or pnpm

### Installation

```bash
# Enter frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Open http://localhost:5173
```

### Build for Production

```bash
# Type check and build
npm run check
npm run build

# Preview production build
npm run preview
```

## Project Structure

```
frontend/
├── src/
│   ├── lib/
│   │   ├── api.ts        # API client and types
│   │   ├── auth.ts       # Authentication utilities
│   │   └── stores.ts     # Svelte stores (state management)
│   ├── routes/           # SvelteKit routes (file-based routing)
│   │   ├── +page.svelte              # Home / Dashboard
│   │   ├── login/+page.svelte        # Login page
│   │   ├── receipts/                 # Receipt management
│   │   │   ├── +page.svelte          # Receipt list
│   │   │   ├── new/+page.svelte      # Create receipt
│   │   │   └── [id]/+page.svelte     # Receipt detail
│   │   ├── tags/+page.svelte         # Tag management
│   │   ├── settings/                 # Settings pages
│   │   │   ├── +page.svelte          # General settings
│   │   │   └── budgets/+page.svelte  # Budget management
│   │   └── analytics/                # Analytics pages
│   │       ├── +page.svelte          # Overview
│   │       ├── monthly/+page.svelte   # Monthly trends
│   │       ├── by-tag/+page.svelte   # Tag breakdown
│   │       └── by-shop/+page.svelte  # Shop breakdown
│   ├── components/       # Reusable components
│   │   ├── Navbar.svelte   # Navigation component
│   │   └── ...
│   ├── app.html          # HTML template
│   └── app.css           # Global styles
├── static/               # Static assets
├── build/                # Production build output
└── package.json
```

## Development

### Available Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start development server with hot reload |
| `npm run build` | Build for production |
| `npm run preview` | Preview production build locally |
| `npm run check` | Type-check with svelte-check |
| `npm run check:watch` | Type-check in watch mode |

### API Configuration

The API client is configured in `src/lib/api.ts`. By default, it uses relative URLs (`/api/v1`) which work with the Vite proxy in development.

For production, ensure the backend API is accessible at the same origin or configure a reverse proxy.

### Authentication

The app uses JWT tokens stored in memory with automatic refresh:
- Access token: Short-lived (15 min), stored in auth store
- Refresh token: Long-lived (7 days), stored in httpOnly cookie

Authentication state is managed in `src/lib/auth.ts`.

### State Management

Primary stores in `src/lib/stores.ts`:
- `authStore` - User authentication state
- `toastStore` - Toast notification queue
- `dateRangeStore` - Global date range filter
- `analyticsDateRangeStore` - Analytics-specific date range
- `pendingCountStore` - Pending receipts count

## Routes & Pages

| Route | Description |
|-------|-------------|
| `/` | Dashboard with summary and recent receipts |
| `/login` | User login page |
| `/receipts` | List all receipts with filters |
| `/receipts/new` | Create new receipt manually |
| `/receipts/[id]` | View/edit receipt details |
| `/tags` | Manage tags |
| `/settings` | User settings & notifications |
| `/settings/budgets` | Budget management |
| `/analytics` | Analytics overview |
| `/analytics/monthly` | Monthly spending trends |
| `/analytics/by-tag` | Spending by tag with budget progress |
| `/analytics/by-shop` | Spending by shop |
| `/queue` | Pending receipts queue |

## PWA Features

The app is configured as a Progressive Web App:
- **Offline Support**: Service worker caches assets
- **Installable**: Can be added to home screen on mobile
- **Standalone**: Runs in full-screen mode when installed

### Icons

Place icons in `static/icons/`:
- `icon-192x192.png` - For home screen
- `icon-512x512.png` - For splash screens

### Manifest

PWA manifest is configured in `vite.config.ts`:
- Theme color: `#2563eb` (blue)
- Background: `#ffffff`
- Display: `standalone`

## Styling

### Tailwind CSS

Uses Tailwind CSS 4 with the Vite plugin. Configuration is minimal - just use Tailwind classes directly.

Common patterns:
```html
<div class="bg-white rounded-lg shadow-md p-4">
  <h2 class="text-lg font-semibold text-gray-900">Title</h2>
  <p class="text-gray-600 mt-2">Content</p>
</div>
```

### Color Scheme

- Primary: Blue (`#2563eb`)
- Success: Green
- Warning: Yellow/Amber
- Error: Red
- Background: Gray-50 to Gray-100

## API Client

The API client in `src/lib/api.ts` provides:
- TypeScript interfaces for all data types
- Methods for all backend endpoints
- Automatic JSON parsing
- Error handling with custom ApiError class

Example usage:
```typescript
import { api } from '$lib/api';

// Fetch receipts
const response = await api.receipts.list({ page: 1, limit: 20 });

// Create budget
await api.budgets.create({
  tag_id: 'uuid-here',
  month: '2026-04',
  amount_limit: 500000
});
```

## Development Guidelines

### Adding a New Page

1. Create directory in `src/routes/` (e.g., `src/routes/new-feature/+page.svelte`)
2. Implement page component with `<script lang="ts">`
3. Add link to `Navbar.svelte` if needed
4. Use existing pages as templates

### Adding API Integration

1. Add interface to `src/lib/api.ts` if new type
2. Add API method to the appropriate class section
3. Use in component with `await`
4. Handle loading and error states

### Component Structure

```svelte
<script lang="ts">
  // Imports
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  
  // State
  let data: DataType[] = [];
  let loading = true;
  let error: string | null = null;
  
  // Load data
  async function load() {
    try {
      data = await api.endpoint.method();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
  
  onMount(load);
</script>

<!-- Template -->
{#if loading}
  <div>Loading...</div>
{:else if error}
  <div class="text-red-600">{error}</div>
{:else}
  <div><!-- Content --></div>
{/if}
```

## Build & Deploy

### Static Build

The app builds to static files suitable for any static host:

```bash
npm run build
# Output in build/
```

### Docker

```bash
# Build image
docker build -t receipt-manager-frontend .

# Run container
docker run -p 3000:3000 receipt-manager-frontend
```

### Environment Variables

The frontend uses minimal environment configuration:

| Variable | Description |
|----------|-------------|
| `PUBLIC_API_URL` | Backend API URL (default: relative) |

## Troubleshooting

### Build Errors

```bash
# Clear SvelteKit cache
rm -rf .svelte-kit

# Reinstall dependencies
rm -rf node_modules package-lock.json
npm install
```

### Type Errors

```bash
# Run type checker
npm run check

# Watch mode for development
npm run check:watch
```

### PWA Not Updating

- Unregister service worker in DevTools
- Clear site data
- Hard refresh (Ctrl+Shift+R)

## Browser Support

- Chrome/Edge 90+
- Firefox 90+
- Safari 15+
- Mobile Safari (iOS 15+)
- Chrome Android

## License

MIT
