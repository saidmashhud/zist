# Zist Architecture

## Service Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                         GATEWAY (:8000 / :8443)                      │
│                                                                      │
│  1. Strip inbound X-User-* headers                                   │
│  2. Read zist_session cookie                                         │
│  3. Validate JWT:                                                    │
│     ├─ Fast: JWKS cache → crypto verify (RS256 / ES256)              │
│     └─ Fallback: HTTP → mgID /v1/auth/validate                      │
│  4. Inject: X-User-ID, X-Tenant-ID, X-User-Email, X-User-Scopes    │
│  5. Route by path prefix (strip /api before forwarding)              │
│                                                                      │
│  /api/auth/*           → local OIDC handler (PKCE)                   │
│  /api/listings/*       → LISTINGS (:8001)                            │
│  /api/bookings/*       → BOOKINGS (:8002)                            │
│  /api/payments/*       → PAYMENTS (:8003)                            │
│  /api/admin/webhooks/* → mgEvents SDK (scope: zist.webhooks.manage)  │
│  /*                    → WEB (:3000, SvelteKit)                      │
└──────────┬──────────────────┬──────────────────┬─────────────────────┘
           │                  │                  │
           ▼                  ▼                  ▼
   ┌──────────────┐  ┌──────────────┐   ┌──────────────┐
   │   LISTINGS   │  │   BOOKINGS   │   │   PAYMENTS   │
   │   :8001      │  │   :8002      │   │   :8003      │
   │              │  │              │   │              │
   │ CRUD for     │  │ Booking CRUD │   │ POST /checkout│
   │ property     │  │ + status     │   │ POST /webhooks│
   │ listings     │  │ transitions  │   │   /mashgate   │
   │              │  │              │   │              │
   │ Scope:       │  │ Scope:       │   │ Scope:       │
   │ listings.*   │  │ bookings.*   │   │ payments.*   │
   └──────┬───────┘  └──────┬───────┘   └──────┬───────┘
          │                  │                  │
          └────────┬─────────┘                  │
                   ▼                            │
            ┌────────────┐                      │
            │ PostgreSQL │◄─────────────────────┘
            │   :5433    │    (dedup table)
            └────────────┘
```

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
