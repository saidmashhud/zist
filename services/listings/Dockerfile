# Build context: zist/ root (context: . in docker-compose)
FROM golang:1.24-alpine AS build
WORKDIR /workspace

# Copy internal auth module (replace directive target)
COPY internal/auth /workspace/auth
COPY internal/httputil /workspace/httputil

# Copy listings service
COPY services/listings /workspace/listings

WORKDIR /workspace/listings
RUN printf 'go 1.24\nuse .\nreplace github.com/saidmashhud/zist/internal/auth => /workspace/auth\nreplace github.com/saidmashhud/zist/internal/httputil => /workspace/httputil\n' > go.work
RUN go mod download
RUN CGO_ENABLED=0 go build -o /listings .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /listings /listings
EXPOSE 8001
ENTRYPOINT ["/listings"]
