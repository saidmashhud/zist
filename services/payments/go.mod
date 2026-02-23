module github.com/saidmashhud/zist/services/payments

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/saidmashhud/mashgate/packages/sdk/go v0.0.0
	github.com/saidmashhud/zist/internal/auth v0.0.0
)

// In a Go workspace (go.work), the replace in go.work takes precedence.
// This replace is also needed for standalone builds (e.g. Docker without workspace).
replace github.com/saidmashhud/mashgate/packages/sdk/go => ../../../mashgate/packages/sdk/go

replace github.com/saidmashhud/zist/internal/auth => ../../internal/auth
