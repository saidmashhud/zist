#!/usr/bin/env bash
# Zist E2E smoke test — runs against a live docker-compose stack.
# Tests public endpoints + auth enforcement (no auth token needed).
#
# Usage: ./tests/e2e/smoke.sh
# Environment:
#   GATEWAY_URL    (default: http://localhost:8000)
set -euo pipefail

GATEWAY="${GATEWAY_URL:-http://localhost:8000}"
PASS=0; FAIL=0

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
info()  { printf '\033[36m%s\033[0m\n' "$*"; }

ok()   { PASS=$((PASS+1)); green "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); red   "  FAIL: $1 — $2"; }

check() {
  local name="$1"; local actual="$2"; local expected="$3"
  if echo "$actual" | grep -q "$expected"; then
    ok "$name"
  else
    fail "$name" "expected '$expected' in: $actual"
  fi
}

check_status() {
  local name="$1"; local actual="$2"; local expected="$3"
  if [[ "$actual" == "$expected" ]]; then
    ok "$name"
  else
    fail "$name" "expected status $expected, got $actual"
  fi
}

# Helper: get HTTP status code (no -f so curl always outputs)
status() {
  curl -s -o /dev/null -w "%{http_code}" "$@" 2>/dev/null
}

info "=== Zist Smoke Tests ==="
info "Gateway: $GATEWAY"
echo

# ── Health check ──────────────────────────────────────────────────────────
info "--- Health ---"
check_status "GET /healthz" "$(status "$GATEWAY/healthz")" "200"

# ── Public read endpoints ─────────────────────────────────────────────────
info "--- Public reads ---"
LISTINGS=$(curl -s "$GATEWAY/api/listings" 2>/dev/null || echo '{}')
check "GET /api/listings returns JSON" "$LISTINGS" '"listings"'

# ── Auth enforcement (writes require JWT, must return 401) ────────────────
info "--- Auth enforcement (no auth → 401) ---"
check_status "POST /api/listings → 401" "$(status -X POST "$GATEWAY/api/listings" -H 'Content-Type: application/json' -d '{}')" "401"
check_status "POST /api/bookings → 401" "$(status -X POST "$GATEWAY/api/bookings" -H 'Content-Type: application/json' -d '{}')" "401"
check_status "DELETE /api/listings/nonexistent → 401" "$(status -X DELETE "$GATEWAY/api/listings/nonexistent")" "401"

# ── Webhook scope enforcement ─────────────────────────────────────────────
info "--- Webhook scope (no auth → 401) ---"
check_status "GET /api/admin/webhooks → 401" "$(status "$GATEWAY/api/admin/webhooks")" "401"

# ── Auth routes exist ─────────────────────────────────────────────────────
info "--- Auth routes ---"
LOGIN_STATUS=$(status -X POST "$GATEWAY/api/auth/login" -H 'Content-Type: application/json' -d '{"email":"nobody@example.com","password":"wrong"}')
# Route should exist; acceptable statuses depend on backend/auth config.
if [[ "$LOGIN_STATUS" != "404" ]]; then
  ok "POST /api/auth/login exists (status $LOGIN_STATUS)"
else
  fail "POST /api/auth/login" "got 404 — route not registered"
fi

check_status "GET /api/auth/me → 401" "$(status "$GATEWAY/api/auth/me")" "401"

# ── Frontend serving ──────────────────────────────────────────────────────
info "--- Frontend ---"
check_status "GET / → 200 (SvelteKit)" "$(status "$GATEWAY/")" "200"

# ── Summary ───────────────────────────────────────────────────────────────
echo
info "=== Results: $PASS passed, $FAIL failed ==="
[[ $FAIL -eq 0 ]]
