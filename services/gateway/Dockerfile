# Build context: personal/ root (context: .. in docker-compose)
# Gateway needs both Mashgate SDK and internal/auth
FROM golang:1.24-alpine AS build
WORKDIR /workspace

# Copy Mashgate SDK (local replace directive target)
COPY mashgate/packages/sdk-go /workspace/mashgate-sdk

# Copy internal auth module
COPY zist/internal/auth /workspace/auth
COPY zist/internal/httputil /workspace/httputil

# Copy gateway service
COPY zist/services/gateway /workspace/gateway

WORKDIR /workspace/gateway
RUN printf 'go 1.24\nuse .\nreplace github.com/saidmashhud/mashgate/packages/sdk-go => /workspace/mashgate-sdk\nreplace github.com/saidmashhud/zist/internal/auth => /workspace/auth\nreplace github.com/saidmashhud/zist/internal/httputil => /workspace/httputil\n' > go.work
RUN go mod download
RUN CGO_ENABLED=0 go build -o /gateway .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /gateway /gateway
# 8000 = HTTP/1.1+2, 8443 = HTTP/3 (QUIC/UDP)
EXPOSE 8000 8443/udp
ENTRYPOINT ["/gateway"]
