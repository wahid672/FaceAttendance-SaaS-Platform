# ==========================================
# Stage 1: Build binary using Go on Alpine
# ==========================================
FROM golang:1.24-alpine AS builder

# Install build essentials and certs
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked lean binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# ==========================================
# Stage 2: Minimal runtime image using Alpine
# ==========================================
FROM alpine:3.20

# Install runtime certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

USER appuser

# Expose Go API port
EXPOSE 8080

# Health check probe
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/server"]
