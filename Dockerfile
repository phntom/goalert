# Build stage: pinned Go toolchain matching go.mod.
FROM golang:1.27.4-alpine3.24 AS builder
WORKDIR /app
# Cache dependencies first; go.mod/go.sum are authoritative (never `go mod tidy` in a build).
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /app/alert ./cmd/goalert-bot

# Runtime stage: distroless static, non-root, no shell.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder --chown=65532:65532 /app/alert /app/alert
USER nonroot:nonroot
EXPOSE 3000
CMD ["/app/alert"]
