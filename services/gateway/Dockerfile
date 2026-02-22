FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o gateway .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /app/gateway /gateway
# 8000 = HTTP/1.1+2, 8443 = HTTP/3 (QUIC/UDP)
EXPOSE 8000 8443/udp
ENTRYPOINT ["/gateway"]
