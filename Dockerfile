# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/formation ./cmd/api

# ── Run stage ─────────────────────────────────────────────────────────────────
FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/formation .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./formation"]
