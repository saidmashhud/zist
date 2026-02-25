# Build context: personal/ root (context: .. in docker-compose)
# Payments needs Mashgate SDK + internal/auth + internal/dedup
FROM golang:1.24-alpine AS build
RUN apk add --no-cache git
WORKDIR /workspace

# Copy Mashgate SDK (local replace directive target)
COPY mashgate/sdk/go /workspace/mashgate-sdk

# Copy internal modules
COPY zist/internal/auth /workspace/auth
COPY zist/internal/dedup /workspace/dedup
COPY zist/internal/httputil /workspace/httputil

# Copy payments service
COPY zist/services/payments /workspace/payments

# Create a go.work that wires all local replaces
WORKDIR /workspace/payments
RUN printf 'go 1.24\nuse .\nreplace github.com/saidmashhud/mashgate/packages/sdk-go => /workspace/mashgate-sdk\nreplace github.com/saidmashhud/zist/internal/auth => /workspace/auth\nreplace github.com/saidmashhud/zist/internal/dedup => /workspace/dedup\nreplace github.com/saidmashhud/zist/internal/httputil => /workspace/httputil\n' > go.work
RUN GOPROXY=direct go mod download
RUN CGO_ENABLED=0 go build -o /payments .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /payments /payments
EXPOSE 8003
ENTRYPOINT ["/payments"]
