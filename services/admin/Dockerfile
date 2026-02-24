# Build context: zist/ root
FROM golang:1.24-alpine AS build
WORKDIR /workspace

COPY internal/auth /workspace/auth
COPY internal/httputil /workspace/httputil
COPY services/admin /workspace/admin

WORKDIR /workspace/admin
RUN printf 'go 1.24\nuse .\nreplace github.com/saidmashhud/zist/internal/auth => /workspace/auth\nreplace github.com/saidmashhud/zist/internal/httputil => /workspace/httputil\n' > go.work
RUN go mod download
RUN CGO_ENABLED=0 go build -o /admin .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /admin /admin
EXPOSE 8005
ENTRYPOINT ["/admin"]
