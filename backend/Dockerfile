# Stage 1: Builder
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /build/api ./cmd/api

# Stage 2: Runtime
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/api .

# Use non-root user
USER appuser

# Document exposed port
EXPOSE 8080

ENTRYPOINT ["./api"]
