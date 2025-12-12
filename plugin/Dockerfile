# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the plugin binary (arm64 for Scaleway K8s)
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o /popsigner-secp256k1 \
    ./cmd/plugin

# Final stage - minimal image with just the binary
FROM alpine:3.19

LABEL org.opencontainers.image.title="POPSigner secp256k1 Plugin"
LABEL org.opencontainers.image.description="OpenBao plugin for secp256k1 signing"

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /popsigner-secp256k1 /usr/local/bin/popsigner-secp256k1

# Make executable
RUN chmod +x /usr/local/bin/popsigner-secp256k1

# The plugin binary - OpenBao will load this
ENTRYPOINT ["/usr/local/bin/popsigner-secp256k1"]
