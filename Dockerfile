# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies including gcc for CGO
RUN apk add --no-cache \
    git \
    gcc \
    g++ \
    musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled and optimizations
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -a \
    -o pgschema .

# Final stage - Alpine for debugging capabilities
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    libc6-compat && \
    adduser -D -g '' pgschema

# Copy binary from builder
COPY --from=builder /build/pgschema /usr/local/bin/pgschema

# Switch to non-root user
USER pgschema

# Set entrypoint
ENTRYPOINT ["pgschema"]