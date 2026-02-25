# Build context: zist/ root (context: . in docker-compose)
FROM golang:1.24-alpine AS build
RUN apk add --no-cache git
WORKDIR /workspace

COPY internal/auth /workspace/auth
COPY internal/httputil /workspace/httputil
COPY services/reviews /workspace/reviews

WORKDIR /workspace/reviews
RUN printf 'go 1.24\nuse .\nreplace github.com/saidmashhud/zist/internal/auth => /workspace/auth\nreplace github.com/saidmashhud/zist/internal/httputil => /workspace/httputil\n' > go.work
RUN GOPROXY=direct go mod download
RUN CGO_ENABLED=0 go build -o /reviews .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /reviews /reviews
EXPOSE 8004
ENTRYPOINT ["/reviews"]
