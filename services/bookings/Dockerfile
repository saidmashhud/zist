# Build context: zist/ root (context: . in docker-compose)
FROM golang:1.24-alpine AS build
RUN apk add --no-cache git
WORKDIR /workspace

# Copy internal auth module (replace directive target)
COPY internal/auth /workspace/auth
COPY internal/httputil /workspace/httputil

# Copy bookings service
COPY services/bookings /workspace/bookings

WORKDIR /workspace/bookings
RUN printf 'go 1.24\nuse .\nreplace github.com/saidmashhud/zist/internal/auth => /workspace/auth\nreplace github.com/saidmashhud/zist/internal/httputil => /workspace/httputil\n' > go.work
RUN GOPROXY=direct go mod download
RUN CGO_ENABLED=0 go build -o /bookings .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /bookings /bookings
EXPOSE 8002
ENTRYPOINT ["/bookings"]
