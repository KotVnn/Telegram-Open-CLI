# Docker Deployment Example

This example demonstrates deploying TOC using Docker.

## Quick Start

### 1. Create Docker Compose File

```yaml
version: '3.8'

services:
  toc:
    build: .
    container_name: toc
    restart: unless-stopped
    volumes:
      - toc-data:/home/toc/.toc
    environment:
      - TOC_TELEGRAM_TOKEN=${TOC_TELEGRAM_TOKEN}
    ports:
      - "9090:9090"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9090/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  toc-data:
```

### 2. Create Environment File

```bash
# .env
TOC_TELEGRAM_TOKEN=your-bot-token-here
```

### 3. Build and Run

```bash
docker-compose up -d
```

## Production Setup

### With Prometheus Monitoring

```yaml
version: '3.8'

services:
  toc:
    build: .
    container_name: toc
    restart: unless-stopped
    volumes:
      - toc-data:/home/toc/.toc
      - ./config.toml:/home/toc/.toc/config.toml:ro
    environment:
      - TOC_TELEGRAM_TOKEN=${TOC_TELEGRAM_TOKEN}
    ports:
      - "9090:9090"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9090/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    container_name: toc-prometheus
    restart: unless-stopped
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    ports:
      - "9091:9090"

  grafana:
    image: grafana/grafana:latest
    container_name: toc-grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana-data:/var/lib/grafana
    ports:
      - "3000:3000"
    depends_on:
      - prometheus

volumes:
  toc-data:
  prometheus-data:
  grafana-data:
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'toc'
    static_configs:
      - targets: ['toc:9090']
```

### Grafana Dashboard

Import the TOC dashboard using:
- Dashboard ID: (create and share your dashboard)
- Metrics endpoint: `http://toc:9090/metrics`

## Commands

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# View logs
docker-compose logs -f toc

# Restart
docker-compose restart toc

# Update
docker-compose pull
docker-compose up -d
```

## Backup

```bash
# Backup database
docker-compose exec toc sqlite3 /home/toc/.toc/toc.db .backup /home/toc/.toc/backup.db
docker cp toc:/home/toc/.toc/backup.db ./backup-$(date +%Y%m%d).db
```

## Troubleshooting

### Check Container Status

```bash
docker-compose ps
docker-compose logs toc
```

### Access Container Shell

```bash
docker-compose exec toc sh
```

### View Metrics

```bash
curl http://localhost:9090/metrics
```
