# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/app

# Download golang-migrate (pinned to v4.17.1 for Go 1.23 compatibility)
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# Create entrypoint script
RUN echo '#!/bin/sh' > /app/entrypoint.sh && \
    echo 'set -e' >> /app/entrypoint.sh && \
    echo 'echo "Running database migrations..."' >> /app/entrypoint.sh && \
    echo 'migrate -path /app/migrations -database "$DATABASE_URL" up' >> /app/entrypoint.sh && \
    echo 'echo "Migrations complete. Starting server..."' >> /app/entrypoint.sh && \
    echo 'exec ./server' >> /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 3000

ENTRYPOINT ["/app/entrypoint.sh"]