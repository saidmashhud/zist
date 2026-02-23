#!/usr/bin/env bash
# provision-mashgate-dev.sh
#
# Idempotent dev-environment bootstrap for Zist <-> Mashgate integration.
# Sets up: tenant, OAuth client, app permissions/scopes, and default roles.
#
# Usage:
#   MGID_URL=http://localhost:9661 \
#   MGID_ADMIN_TOKEN=<admin-token> \
#   ./provision-mashgate-dev.sh
#
# Optional overrides:
#   TENANT_NAME   (default: "Zist Dev")
#   TENANT_EMAIL  (default: "admin@zist.local")
#   CLIENT_ID     (default: "zist-local")
#   CLIENT_SECRET (default: auto-generated)

set -euo pipefail

# ─── Config ───────────────────────────────────────────────────────────────────

MGID_URL="${MGID_URL:-http://localhost:9661}"
MGID_ADMIN_TOKEN="${MGID_ADMIN_TOKEN:?MGID_ADMIN_TOKEN is required}"
TENANT_NAME="${TENANT_NAME:-Zist Dev}"
TENANT_EMAIL="${TENANT_EMAIL:-admin@zist.local}"
CLIENT_ID="${CLIENT_ID:-zist-local}"
CLIENT_SECRET="${CLIENT_SECRET:-$(openssl rand -hex 16)}"
REDIRECT_URI="${REDIRECT_URI:-http://localhost:8000/api/auth/callback}"

AUTH_HEADER="Authorization: Bearer $MGID_ADMIN_TOKEN"

log() { echo "[provision] $*"; }
err() { echo "[provision] ERROR: $*" >&2; }

# ─── Helper: POST JSON ────────────────────────────────────────────────────────

post_json() {
  local path="$1"
  local data="$2"
  curl -sS -w "\n%{http_code}" \
    -X POST \
    -H "$AUTH_HEADER" \
    -H "Content-Type: application/json" \
    -d "$data" \
    "$MGID_URL$path"
}

get_json() {
  local path="$1"
  curl -sS \
    -H "$AUTH_HEADER" \
    "$MGID_URL$path"
}

# ─── Step 1: Create / find tenant ─────────────────────────────────────────────

log "Step 1: Ensure tenant exists..."

TENANT_RESPONSE=$(post_json "/v1/tenants" "{
  \"name\": \"$TENANT_NAME\",
  \"email\": \"$TENANT_EMAIL\"
}")

HTTP_CODE=$(echo "$TENANT_RESPONSE" | tail -1)
TENANT_BODY=$(echo "$TENANT_RESPONSE" | head -n -1)

if [[ "$HTTP_CODE" == "201" ]]; then
  TENANT_ID=$(echo "$TENANT_BODY" | grep -o '"tenantId":"[^"]*"' | cut -d'"' -f4)
  log "Tenant created: $TENANT_ID"
elif [[ "$HTTP_CODE" == "409" || "$HTTP_CODE" == "422" ]]; then
  # Tenant already exists — find by email
  TENANTS=$(get_json "/v1/tenants?email=$TENANT_EMAIL")
  TENANT_ID=$(echo "$TENANTS" | grep -o '"tenantId":"[^"]*"' | head -1 | cut -d'"' -f4)
  log "Tenant already exists: $TENANT_ID"
else
  err "Unexpected status $HTTP_CODE creating tenant: $TENANT_BODY"
  exit 1
fi

if [[ -z "$TENANT_ID" ]]; then
  err "Could not determine tenant ID"
  exit 1
fi

# ─── Step 2: Create / update OAuth client ─────────────────────────────────────

log "Step 2: Ensure OAuth client '$CLIENT_ID' exists..."

CLIENT_RESPONSE=$(post_json "/v1/oidc/clients" "{
  \"clientId\": \"$CLIENT_ID\",
  \"clientSecret\": \"$CLIENT_SECRET\",
  \"tenantId\": \"$TENANT_ID\",
  \"redirectUris\": [\"$REDIRECT_URI\"],
  \"grantTypes\": [\"authorization_code\", \"refresh_token\"],
  \"responseTypes\": [\"code\"],
  \"pkceRequired\": true,
  \"name\": \"Zist (local dev)\"
}")

HTTP_CODE=$(echo "$CLIENT_RESPONSE" | tail -1)
CLIENT_BODY=$(echo "$CLIENT_RESPONSE" | head -n -1)

if [[ "$HTTP_CODE" == "201" || "$HTTP_CODE" == "200" || "$HTTP_CODE" == "409" ]]; then
  log "OAuth client ready: $CLIENT_ID"
else
  err "Failed to create OAuth client (HTTP $HTTP_CODE): $CLIENT_BODY"
  exit 1
fi

# ─── Step 3: Register app permissions (scopes) ────────────────────────────────

log "Step 3: Registering app permissions..."

SCOPES=(
  "zist.listings.read:Read rental listings"
  "zist.listings.manage:Create and manage listings"
  "zist.bookings.read:Read booking records"
  "zist.bookings.manage:Create and manage bookings"
  "zist.payments.create:Initiate checkout sessions"
  "zist.payments.read:Read payment records"
  "zist.webhooks.manage:Manage webhook endpoints"
)

for SCOPE_DEF in "${SCOPES[@]}"; do
  SCOPE_NAME="${SCOPE_DEF%%:*}"
  SCOPE_DESC="${SCOPE_DEF##*:}"

  RESULT=$(post_json "/v1/iam/app-scopes" "{
    \"clientId\": \"$CLIENT_ID\",
    \"scope\": \"$SCOPE_NAME\",
    \"description\": \"$SCOPE_DESC\"
  }")
  HTTP_CODE=$(echo "$RESULT" | tail -1)
  if [[ "$HTTP_CODE" == "201" || "$HTTP_CODE" == "200" || "$HTTP_CODE" == "409" ]]; then
    log "  ✓ $SCOPE_NAME"
  else
    err "  Failed to register $SCOPE_NAME (HTTP $HTTP_CODE): $(echo "$RESULT" | head -n -1)"
  fi
done

# ─── Step 4: Create default roles ─────────────────────────────────────────────

log "Step 4: Creating default roles..."

declare -A ROLE_SCOPES
ROLE_SCOPES["zist_admin"]="zist.listings.read,zist.listings.manage,zist.bookings.read,zist.bookings.manage,zist.payments.create,zist.payments.read,zist.webhooks.manage"
ROLE_SCOPES["zist_operator"]="zist.listings.manage,zist.bookings.manage,zist.payments.create,zist.bookings.read,zist.listings.read"
ROLE_SCOPES["zist_viewer"]="zist.listings.read,zist.bookings.read,zist.payments.read"

for ROLE_NAME in zist_admin zist_operator zist_viewer; do
  ROLE_RESULT=$(post_json "/v1/iam/roles" "{
    \"name\": \"$ROLE_NAME\",
    \"tenantId\": \"$TENANT_ID\",
    \"description\": \"Zist built-in role\"
  }")
  HTTP_CODE=$(echo "$ROLE_RESULT" | tail -1)
  ROLE_BODY=$(echo "$ROLE_RESULT" | head -n -1)

  ROLE_ID=$(echo "$ROLE_BODY" | grep -o '"roleId":"[^"]*"' | cut -d'"' -f4)
  if [[ -z "$ROLE_ID" ]]; then
    ROLE_ID=$(echo "$ROLE_BODY" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
  fi

  if [[ "$HTTP_CODE" == "201" || "$HTTP_CODE" == "200" ]]; then
    log "  ✓ Role created: $ROLE_NAME ($ROLE_ID)"
  elif [[ "$HTTP_CODE" == "409" ]]; then
    log "  ~ Role exists: $ROLE_NAME"
  else
    err "  Failed to create role $ROLE_NAME (HTTP $HTTP_CODE)"
    continue
  fi

  # Grant scopes to role
  if [[ -n "$ROLE_ID" ]]; then
    IFS=',' read -ra SCOPE_LIST <<< "${ROLE_SCOPES[$ROLE_NAME]}"
    for SCOPE in "${SCOPE_LIST[@]}"; do
      GRANT_RESULT=$(post_json "/v1/iam/roles/$ROLE_ID/scopes" "{
        \"scope\": \"$SCOPE\",
        \"clientId\": \"$CLIENT_ID\"
      }")
      GRANT_CODE=$(echo "$GRANT_RESULT" | tail -1)
      if [[ "$GRANT_CODE" == "200" || "$GRANT_CODE" == "201" || "$GRANT_CODE" == "204" || "$GRANT_CODE" == "409" ]]; then
        log "    ✓ Granted $SCOPE → $ROLE_NAME"
      else
        log "    ~ Could not grant $SCOPE → $ROLE_NAME (HTTP $GRANT_CODE)"
      fi
    done
  fi
done

# ─── Output .env values ───────────────────────────────────────────────────────

echo ""
echo "════════════════════════════════════════"
echo "  Zist dev environment is ready!"
echo "════════════════════════════════════════"
echo ""
echo "Add to your .env (or docker-compose env):"
echo ""
echo "  MGID_URL=$MGID_URL"
echo "  MGID_CLIENT_ID=$CLIENT_ID"
echo "  MGID_CLIENT_SECRET=$CLIENT_SECRET"
echo "  MGID_REDIRECT_URI=$REDIRECT_URI"
echo "  TENANT_ID=$TENANT_ID"
echo ""
echo "Roles created: zist_admin, zist_operator, zist_viewer"
echo "Scopes registered under client: $CLIENT_ID"
