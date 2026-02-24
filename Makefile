.PHONY: build test test-e2e test-e2e-web test-unit smoke lint clean docker-up docker-down \
	ops-up ops-down prod-gate dr-drill release-baseline

# ── Build ──────────────────────────────────────────────────────────────────

build:
	go build ./services/gateway
	go build ./services/listings
	go build ./services/bookings
	go build ./services/payments

# ── Tests ──────────────────────────────────────────────────────────────────

test: test-unit test-e2e test-e2e-web

test-unit:
	go test ./internal/auth/... ./internal/dedup/... ./internal/httputil/... ./internal/mashgate/... \
		./services/gateway/... ./services/listings/... ./services/bookings/... ./services/payments/... \
		-v -count=1

test-e2e:
	cd tests/e2e && INTERNAL_TOKEN=$${INTERNAL_TOKEN:-dev-internal-token} MASHGATE_WEBHOOK_SECRET=$${MASHGATE_WEBHOOK_SECRET:-dev-whsec} go test -v -count=1 -timeout 120s ./...

test-e2e-web:
	cd apps/web && npm run test:e2e

smoke:
	bash tests/e2e/smoke.sh

prod-gate:
	python3 tests/ops/prod-gate.py

dr-drill:
	python3 tests/ops/dr-drill.py

# ── Lint ───────────────────────────────────────────────────────────────────

lint:
	go vet ./internal/auth/... ./internal/dedup/... ./internal/httputil/... ./internal/mashgate/... \
		./services/gateway/... ./services/listings/... ./services/bookings/... ./services/payments/...

# ── Docker ─────────────────────────────────────────────────────────────────

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

ops-up:
	docker compose -f docker-compose.yml -f ops/docker-compose.ops.yml up -d blackbox prometheus tempo otel-collector grafana

ops-down:
	docker compose -f docker-compose.yml -f ops/docker-compose.ops.yml down

release-baseline:
	bash scripts/release/tag-baseline.sh

# ── Clean ──────────────────────────────────────────────────────────────────

clean:
	go clean ./...
