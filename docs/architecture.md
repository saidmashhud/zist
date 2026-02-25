# Zist Architecture

## Service Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                         GATEWAY (:8000 / :8443)                      │
│                                                                      │
│  1. Strip inbound X-User-* headers                                   │
│  2. Read zist_session cookie                                         │
│  3. Validate JWT (JWKS cache → RS256/ES256, fallback mgID HTTP)      │
│  4. Inject: X-User-ID, X-Tenant-ID, X-User-Email, X-User-Scopes    │
│  5. Route by path prefix (strip /api before forwarding)              │
│                                                                      │
│  /api/auth/*           → local OIDC handler (PKCE)                   │
│  /api/listings/*       → LISTINGS (:8001)                            │
│  /api/bookings/*       → BOOKINGS (:8002)                            │
│  /api/payments/*       → PAYMENTS (:8003)                            │
│  /api/reviews/*        → REVIEWS  (:8004)                            │
│  /api/admin/*          → ADMIN    (:8005)                            │
│  /api/search/*         → SEARCH   (:8006)                            │
│  /api/chat/*           → HookLine WebSocket proxy                    │
│  /api/admin/webhooks/* → mgEvents SDK (scope: zist.webhooks.manage)  │
│  /*                    → WEB (:3000, SvelteKit)                      │
└──────┬─────────────────┬──────────┬──────────────┬───────────────────┘
       │                 │          │              │
       ▼                 ▼          ▼              ▼
┌───────────┐  ┌───────────┐  ┌──────────┐  ┌──────────┐
│ LISTINGS  │  │ BOOKINGS  │  │ PAYMENTS │  │ REVIEWS  │
│ :8001     │  │ :8002     │  │ :8003    │  │ :8004    │
│           │  │           │  │          │  │          │
│ CRUD      │  │ Booking   │  │ Checkout │  │ CRUD     │
│ + search  │  │ state     │  │ Webhooks │  │ Replies  │
│ + avail.  │  │ machine   │  │          │  │ Ratings  │
│           │  │           │  │          │  │          │
│ mgLogs    │  │ mgNotify  │  │ mgPay    │  │ Updates  │
│ analytics │  │ SMS/email │  │ SDK      │  │ listings │
└─────┬─────┘  └─────┬─────┘  └────┬─────┘  └────┬─────┘
      │               │             │              │
      └───────────────┴─────────────┴──────────────┘
                              │
                              ▼
                       ┌────────────┐      ┌──────────┐     ┌──────────┐
                       │ PostgreSQL │      │  ADMIN   │     │  SEARCH  │
                       │   :5433    │◄─────│  :8005   │     │  :8006   │
                       │            │      │ Flags    │     │ Geo      │
                       │ webhook    │      │ Audit    │     │ Filters  │
                       │ dedup      │      │ Tenants  │     │ Sort     │
                       └────────────┘      └──────────┘     └──────────┘
```

## Reviews Service (:8004)
- `POST /reviews` — submit review after completed booking (deduplicated by booking_id)
- `GET /reviews/listing/{id}` — list reviews for a property (public)
- `GET /reviews/my` — list reviews written by authenticated guest
- `POST /reviews/{id}/reply` — host reply to a review
- On create: fires internal `PUT /listings/{id}/rating` to update `average_rating` + `review_count`

### mgNotify Notifications (in Bookings service)
- `notifyClient.NotifyUser(ctx, userID, eventType, msg)` — calls `POST /v1/notify/user`
- Triggered on booking confirmation (fire-and-forget)
- Configured via `MGNOTIFY_URL` + `MASHGATE_API_KEY` env vars

### mgLogs Analytics (in Listings service)
- `analytics.TrackListingView(ctx, tenantID, listingID, hostID)` — on every GET /listings/{id}
- `analytics.TrackSearchQuery(ctx, tenantID, city, guests, resultCount)` — on searches
- Configured via `MGLOGS_URL` + `MASHGATE_API_KEY` env vars

### mgChat WebSocket Proxy (in Gateway)
- `/api/chat` and `/api/chat/*` → HookLine WebSocket (TCP relay)
- Requires auth (X-User-ID injected into upstream connection)
- Enabled when `CHAT_URL` / `HOOKLINE_WS_URL` env var is set

## Admin Service (:8005)
- `GET/POST /admin/flags` — feature flag CRUD (requires `zist.admin` scope)
- `GET /admin/audit` — audit log of admin actions
- `GET/PUT /admin/tenants/{id}` — per-tenant platform configuration

### mgFlags Integration (in Listings service)
- `flags.Client` fetches `/v1/flags` with 30s local cache
- Used for `instant_book_v2`, `search_ranking_ml` feature flags

## Search Service (:8006)

- `GET /search` — full-text and geospatial search with filters (city, lat/lng+radius, dates, guests, price range, amenities, instant book, property type)
- `PUT /search/locations/{id}` — internal endpoint for Listings service to update location index on create/update
- Sort by: `rating`, `price`, `distance`
- Pagination: `limit` + `offset`

### i18n (SvelteKit)
- Supported locales: `ru`, `uz`, `en`, `kk`
- `$lib/i18n` store with `locale` writable and `t` derived store
- `LocaleSwitcher.svelte` component in navbar
- Persisted to `localStorage['zist_locale']`

## Auth Propagation Flow

```
Browser → GET /api/bookings
  │
  ├─ Cookie: zist_session=<JWT>
  │
  ▼
Gateway (propagateAuth middleware)
  │
  ├─ Strip all inbound X-User-* headers
  ├─ Read zist_session cookie value
  ├─ Parse JWT header → extract kid + algorithm
  ├─ JWKS cache lookup (keyed by kid)
  │   ├─ HIT: verify signature locally (crypto/rsa or crypto/ecdsa)
  │   └─ MISS: fetch mgID/.well-known/jwks.json, cache 5 min, retry
  ├─ If JWKS fails → fallback HTTP POST mgID/v1/auth/validate
  ├─ Extract claims: sub, tenant_id, email, scope
  ├─ Set headers:
  │     X-User-ID:     <sub>
  │     X-Tenant-ID:   <tenant_id>
  │     X-User-Email:  <email>
  │     X-User-Scopes: <space-separated scopes>
  │
  ▼
Bookings service
  │
  ├─ auth.Middleware reads X-User-* → stores Principal in ctx
  ├─ RequireScope("zist.bookings.read") → 401/403 if missing
  └─ Handler: p := auth.FromContext(ctx)
```

## Internal Token for Service-to-Service Auth

Booking status transitions (confirm, fail, cancel, checkout) are triggered by the Payments service after processing Mashgate webhooks. These routes are not user-facing — they use a shared `INTERNAL_TOKEN`:

```
Mashgate webhook → Payments service (POST /webhooks/mashgate)
  │
  ├─ Verify signature (HMAC-SHA256)
  ├─ Dedup check (PgStore)
  ├─ Parse event type
  │
  ├─ payment.captured → POST bookings:8002/bookings/{id}/confirm
  │     Header: X-Internal-Token: <INTERNAL_TOKEN>
  │
  └─ payment.failed → POST bookings:8002/bookings/{id}/fail
        Header: X-Internal-Token: <INTERNAL_TOKEN>
```

The `RequireInternalToken` middleware returns 403 if:
- No `X-Internal-Token` header is present
- The token value doesn't match `INTERNAL_TOKEN` env var

## Webhook Dedup

The Payments service deduplicates webhook events to ensure booking status transitions are idempotent:

```
POST /webhooks/mashgate
  │
  ├─ Parse event_id from payload
  ├─ dedup.Check(event_id)
  │   ├─ PostgreSQL: INSERT INTO webhook_dedup ... ON CONFLICT → duplicate
  │   └─ In-memory: sync.Map lookup → duplicate
  │
  ├─ Duplicate → 200 {status: "ok", dedup: "skipped"}
  └─ New → process event → 200 {status: "ok"}
```

**PgStore** (preferred, used when `DATABASE_URL` is set):
- `webhook_dedup` table: `event_id TEXT PK, seen_at TIMESTAMP`
- Atomic check-and-insert via `INSERT ... ON CONFLICT DO NOTHING`
- Background cleanup: hourly, retains 48 hours
- Survives process restarts

**In-memory** (fallback):
- `sync.Map` with TTL-based eviction
- Background cleanup every minute
- Lost on restart

## Scope Enforcement Model

Scopes are checked at two levels:

1. **Gateway level** — webhook admin routes require `zist.webhooks.manage`
2. **Service level** — each route handler declares its required scope

| Scope | Routes |
|-------|--------|
| `zist.listings.read` | GET /listings, GET /listings/:id |
| `zist.listings.manage` | POST /listings, PUT /listings/:id, DELETE /listings/:id |
| `zist.bookings.read` | GET /bookings |
| `zist.bookings.manage` | POST /bookings |
| `zist.payments.create` | POST /checkout |
| `zist.webhooks.manage` | /api/admin/webhooks/* |

Public routes (no auth): GET /listings, GET /listings/:id, GET /bookings/:id, GET /healthz

Internal-only routes (X-Internal-Token): POST /bookings/:id/confirm, /fail, /cancel, PUT /bookings/:id/checkout
