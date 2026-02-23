module github.com/saidmashhud/zist/services/listings

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/saidmashhud/zist/internal/auth v0.0.0
)

replace github.com/saidmashhud/zist/internal/auth => ../../internal/auth
