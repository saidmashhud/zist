FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o listings .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /app/listings /listings
EXPOSE 8001
ENTRYPOINT ["/listings"]
