# Build stage - use Debian to match runtime environment
FROM golang:1.24-bookworm AS builder

# Install build dependencies for CGO (required for go-sqlite3)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o chessflash ./cmd/server

# Runtime stage - use Debian to match Ubuntu binary compatibility
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 chessflash && \
    useradd -u 1000 -g chessflash -m -s /bin/bash chessflash

# Build argument for Stockfish binary path
ARG STOCKFISH_BINARY_PATH=stockfish-ubuntu-x86-64-avx2/stockfish/stockfish-ubuntu-x86-64-avx2

# Copy Stockfish binary
COPY --from=builder /build/${STOCKFISH_BINARY_PATH} /usr/local/bin/stockfish
RUN chmod +x /usr/local/bin/stockfish

# Copy application binary
COPY --from=builder /build/chessflash /app/chessflash

# Copy template files (required at runtime)
COPY --from=builder /build/web/templates /app/web/templates

# Copy static files (CSS, JS, images)
COPY --from=builder /build/web/static /app/web/static

# Create data directory for volume mount
RUN mkdir -p /data && \
    chown -R chessflash:chessflash /data && \
    chown -R chessflash:chessflash /app

WORKDIR /app

USER chessflash

EXPOSE 8080

CMD ["./chessflash"]
