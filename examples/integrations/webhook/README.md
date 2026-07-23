# Webhook Integration Example

This example demonstrates setting up TOC with Telegram webhooks for production use.

## Prerequisites

1. A domain with HTTPS enabled
2. A reverse proxy (nginx, Caddy, etc.)
3. TOC installed and configured

## Setup

### 1. Configure TOC for Webhook Mode

Edit `~/.toc/config.toml`:

```toml
[telegram]
token = "your-bot-token"
mode = "webhook"
webhook_url = "https://yourdomain.com/webhook"
webhook_secret = "your-webhook-secret"
```

### 2. Configure Reverse Proxy

#### Nginx Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /webhook {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /metrics {
        proxy_pass http://localhost:9090;
        # Restrict access to metrics
        allow 127.0.0.1;
        deny all;
    }
}
```

#### Caddy Configuration

```
yourdomain.com {
    reverse_proxy /webhook localhost:8080
    reverse_proxy /metrics localhost:9090 {
        @blocked {
            not remote_ip 127.0.0.1
        }
        respond @blocked 403
    }
}
```

### 3. Start TOC

```bash
toc start
```

### 4. Set Webhook

```bash
curl -X POST https://api.telegram.org/bot<YOUR_TOKEN>/setWebhook \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://yourdomain.com/webhook",
    "secret_token": "your-webhook-secret",
    "allowed_updates": ["message", "callback_query"]
  }'
```

## Verification

### Check Webhook Status

```bash
curl https://api.telegram.org/bot<YOUR_TOKEN>/getWebhookInfo
```

### Test Metrics

```bash
curl http://localhost:9090/metrics
curl http://localhost:9090/health
```

## Docker Compose

For a complete setup with monitoring:

```yaml
version: '3.8'

services:
  toc:
    build: .
    restart: unless-stopped
    volumes:
      - ./config.toml:/home/toc/.toc/config.toml:ro
    environment:
      - TOC_TELEGRAM_TOKEN=${TOC_TELEGRAM_TOKEN}
    ports:
      - "9090:9090"

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - toc

  prometheus:
    image: prom/prometheus:latest
    restart: unless-stopped
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
    ports:
      - "9091:9090"
```

## Security Notes

1. Always use HTTPS in production
2. Use a strong webhook secret
3. Restrict metrics endpoint access
4. Use non-root Docker user
5. Enable authentication in TOC config
