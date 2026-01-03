# Builder stage
FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git ca-certificates curl

# Install golang-migrate
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate /usr/local/bin/migrate

# Copy go.mod first for caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source and build
COPY . .
RUN go mod tidy

# Build main API binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

# Build healthcheck binary (tiny, just checks /api/health)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck

# Runtime stage - using alpine instead of distroless for shell access (needed for migrations)
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /srv

# Copy binaries
COPY --from=build /api /srv/api
COPY --from=build /healthcheck /srv/healthcheck
COPY --from=build /usr/local/bin/migrate /usr/local/bin/migrate

# Copy migrations
COPY migrations /srv/migrations

# Create non-root user
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080

# Default command runs migrations then starts API
CMD ["sh", "-c", "migrate -path /srv/migrations -database \"$DATABASE_URL\" up && /srv/api"]
