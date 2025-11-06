# Builder
FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git ca-certificates

# Copy only go.mod first (go.sum may not exist yet)
COPY go.mod ./
RUN go mod download || true

# Copy the rest, then generate go.sum and build
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

# Runtime (distroless)
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /srv
COPY --from=build /api /srv/api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/srv/api"]
