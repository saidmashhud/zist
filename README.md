# Zist

**Reference rental marketplace** built on the Mashgate ecosystem.

Zist is a full-stack Airbnb-style property rental application that demonstrates how to integrate with Mashgate's products — mgID for authentication, mgPay for payments, and mgEvents for webhook delivery. It serves as the canonical integration example and test harness for the ecosystem.

## Architecture

```
  Browser
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  GATEWAY (:8000 HTTP/1.1+2, :8443 HTTP/3 QUIC)            │
│  ├─ Strips inbound X-User-* headers (injection prevention)  │
│  ├─ Validates zist_session cookie via JWKS (cached) or HTTP  │
│  ├─ Injects X-User-ID, X-Tenant-ID, X-User-Email, Scopes   │
│  └─ Routes:                                                  │
│      /api/auth/*      → local OIDC (PKCE login/callback)    │
│      /api/listings/*  → LISTINGS service                     │
│      /api/bookings/*  → BOOKINGS service                     │
│      /api/payments/*  → PAYMENTS service                     │
│      /api/admin/webhooks/* → mgEvents SDK (scope-gated)      │
│      /*               → SvelteKit frontend                   │
└───────┬─────────┬─────────┬──────────────────────────────────┘
        │         │         │
        ▼         ▼         ▼
   LISTINGS   BOOKINGS   PAYMENTS     WEB (SvelteKit)
    :8001      :8002       :8003        :3000
        │         │         │
        └────┬────┘         │
             ▼              │
         PostgreSQL         │
          :5433             │
                            ▼
                     Mashgate (mgPay, mgEvents)
                            │
                            ▼
                     HookLine (delivery)
```

## Service Map

| Service | Port | Description |
|---------|------|-------------|
| Gateway | 8000 (HTTP), 8443 (HTTP/3) | API routing, OIDC, auth propagation |
| Listings | 8001 | Property listings CRUD |
| Bookings | 8002 | Booking management + status transitions |
| Payments | 8003 | Checkout sessions + Mashgate webhooks |
| Web | 3000 | SvelteKit frontend |
| PostgreSQL | 5433 | Shared database |

## Quick Start

```bash
# 1. Start the stack
docker compose up -d

# 2. (Optional) Provision Mashgate integration
MGID_URL=http://localhost:9661 \
MGID_ADMIN_TOKEN=<admin-token> \
./scripts/provision-mashgate-dev.sh

# 3. Open the app
open http://localhost:8000
```

## Auth Flow

Zist uses **Authorization Code + PKCE** via mgID:

1. `GET /api/auth/login` → redirect to mgID with code_challenge
2. User authenticates on mgID login page
3. `GET /api/auth/callback` → exchange code + verifier for JWT
4. JWT stored in `zist_session` httpOnly cookie (7-day TTL)
5. Gateway validates JWT on each request via **cached JWKS** (5-min refresh, RS256/ES256)
6. Fallback: HTTP call to mgID `/v1/auth/validate` if JWKS fails
7. Headers injected: `X-User-ID`, `X-Tenant-ID`, `X-User-Email`, `X-User-Scopes`

## Security Model

**Header injection prevention** — Gateway strips all inbound `X-User-*` headers before validating and re-setting them.

**Scope enforcement** — Each service route declares required scopes:
- `zist.listings.read` / `zist.listings.manage`
- `zist.bookings.read` / `zist.bookings.manage`
- `zist.payments.create`
- `zist.webhooks.manage`

**Internal token** — Service-to-service calls (Payments → Bookings mutations) use `X-Internal-Token` header. Protects confirm/fail/cancel/checkout routes from external access.

**Webhook dedup** — PostgreSQL-backed dedup store (`webhook_dedup` table) prevents duplicate booking confirmations on at-least-once delivery. Falls back to in-memory store if `DATABASE_URL` is not set.

## Environment Variables

| Variable | Service | Description |
|----------|---------|-------------|
| `GATEWAY_PORT` | Gateway | HTTP port (default: 8000) |
| `GATEWAY_TLS_PORT` | Gateway | HTTP/3 QUIC port (default: 8443) |
| `LISTINGS_URL` | Gateway | Listings service URL |
| `BOOKINGS_URL` | Gateway, Payments | Bookings service URL |
| `PAYMENTS_URL` | Gateway | Payments service URL |
| `WEB_URL` | Gateway | SvelteKit frontend URL |
| `MGID_URL` | Gateway | mgID base URL |
| `MGID_CLIENT_ID` | Gateway | OAuth2 client ID |
| `MGID_CLIENT_SECRET` | Gateway | OAuth2 client secret |
| `MGID_REDIRECT_URI` | Gateway | OAuth2 callback URL |
| `MGID_ADMIN_TOKEN` | Gateway | Admin token for scope bootstrap |
| `MASHGATE_API_KEY` | Gateway, Payments | Mashgate API key |
| `MASHGATE_URL` | Payments | Mashgate base URL |
| `MASHGATE_WEBHOOK_SECRET` | Payments | Webhook signing secret |
| `DATABASE_URL` | Listings, Bookings, Payments | PostgreSQL connection string |
| `INTERNAL_TOKEN` | Bookings, Payments | Service-to-service auth token |
| `SESSION_SECRET` | Gateway | Cookie encryption key |

## Integration with Mashgate

- **mgID**: OIDC provider — handles user auth, issues JWTs with app-scoped permissions
- **mgPay**: Hosted checkout — Payments service creates sessions via Mashgate SDK
- **mgEvents**: Webhook management — Gateway proxies admin endpoints to mg-events (scope-gated)
- **Scope bootstrap**: Gateway registers Zist's app-scoped permissions with mgID at startup (idempotent)

## Testing

```bash
# Unit tests (auth middleware, dedup, JWKS)
make test-unit

# E2E integration tests (requires running services)
make test-e2e

# Smoke tests (requires running docker-compose stack)
make smoke

# Production gate (load + soak + chaos)
LOAD_EVENTS=500 SOAK_SECONDS=300 CHAOS_RESTART=true make prod-gate

# DR + rollback drill (with measured RTO/RPO)
make dr-drill

# All tests
make test
```

## Operations

```bash
# Start observability stack (Prometheus + Alert rules + Grafana + Tempo + OTel Collector)
make ops-up

# Stop observability stack
make ops-down

# Create clean release baseline + git tag (fails on dirty worktree)
make release-baseline
```

## Documentation

- [Architecture](docs/architecture.md) — service diagram, auth flow, internal token, webhook dedup
- [API Reference](docs/api.md) — all endpoints per service with request/response examples
- [SLO / SLI](docs/ops/slo.md) — availability, latency, DR objectives
- [Observability](docs/ops/observability.md) — Prometheus, Grafana, Tempo, OTel setup
- [Incident Runbook](docs/ops/incident-runbook.md) — triage, rollback, restore actions
- [Release Runbook](docs/ops/release-runbook.md) — required gates and baseline tagging
- [Mashgate Integration Guide](../mashgate/docs/zist-integration.md) — end-to-end integration walkthrough
