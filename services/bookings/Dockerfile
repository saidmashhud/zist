FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bookings .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /app/bookings /bookings
EXPOSE 8002
ENTRYPOINT ["/bookings"]
