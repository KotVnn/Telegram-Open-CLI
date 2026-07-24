FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /toc ./cmd/toc

# Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 toc

WORKDIR /home/toc

# Copy binary from builder
COPY --from=builder /toc /usr/local/bin/toc

# Copy example config
COPY configs/example.toml /home/toc/.toc/config.toml

# Set ownership
RUN chown -R toc:toc /home/toc

USER toc

# Volumes for persistent data
VOLUME ["/home/toc/.toc"]

# Metrics port
EXPOSE 9090

ENTRYPOINT ["toc"]
CMD ["start"]
