# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /toc ./cmd/toc

# Runtime stage
# opencode serve requires Node.js; the opencode CLI is installed via npm.
FROM node:22-alpine

RUN apk add --no-cache ca-certificates tzdata git curl \
    && npm install -g opencode-ai@latest \
    && adduser -D -u 1000 toc \
    && mkdir -p /workspace /home/toc/.toc /home/toc/.config/opencode \
    && chown -R toc:toc /workspace /home/toc

WORKDIR /home/toc

COPY --from=builder /toc /usr/local/bin/toc
COPY configs/example.toml /home/toc/.toc/config.toml

USER toc

# Persistent data and the workspace the agent works on
VOLUME ["/home/toc/.toc", "/workspace"]

# Metrics port
EXPOSE 9090

ENTRYPOINT ["toc"]
CMD ["start"]
