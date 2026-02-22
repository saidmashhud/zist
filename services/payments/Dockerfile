# Build context must be set to the root of personal/ so that both zist/ and mashgate/ are available.
# In docker-compose, set: build: { context: ../.., dockerfile: zist/services/payments/Dockerfile }
FROM golang:1.22-alpine AS build
WORKDIR /workspace

# Copy Mashgate SDK (local replace directive target)
COPY mashgate/packages/sdk/go /workspace/mashgate-sdk

# Copy payments service
COPY zist/services/payments /workspace/payments

# Create a go.work that wires the local SDK
WORKDIR /workspace/payments
RUN printf 'go 1.22\nuse .\nreplace github.com/saidmashhud/mashgate/packages/sdk/go => /workspace/mashgate-sdk\n' > /workspace/payments/go.work
RUN go mod download
RUN CGO_ENABLED=0 go build -o /payments .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /payments /payments
EXPOSE 8003
ENTRYPOINT ["/payments"]
