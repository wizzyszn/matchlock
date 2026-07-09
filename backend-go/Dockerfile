# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /matchlock-server ./cmd/keeper

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /matchlock-server .

EXPOSE 8080
ENTRYPOINT ["./matchlock-server"]
