# spa-server

Lightning-fast static file server optimized for Single Page Applications.

An improved fork of [devforth/spa-to-http](https://github.com/devforth/spa-to-http) with additional features and modern Go practices.

## Features

- **Zero Configuration** - Works out of the box for SPAs
- **Tiny Docker Image** - ~5MB using distroless base
- **Health Endpoints** - Built-in `/healthz`, `/health`, `/ready` for Kubernetes
- **Security Headers** - X-Frame-Options, X-Content-Type-Options, etc.
- **CORS Support** - Configurable CORS headers
- **Compression** - Gzip and Brotli pre-compression
- **Graceful Shutdown** - Proper signal handling
- **LRU Caching** - In-memory file caching
- **Optimal Caching** - no-store for HTML, long max-age for assets
- **Custom Fallback** - Configurable fallback file (not just index.html)

## Quick Start

### With Docker (Recommended)

Create a `Dockerfile` in your SPA project:

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM ghcr.io/bawadev/spa-server:latest
COPY --from=builder /app/dist /app
```

Build and run:

```bash
docker build -t my-spa .
docker run -p 8080:8080 my-spa
```

### With Docker Compose + Traefik

```yaml
version: "3.8"

services:
  traefik:
    image: traefik:v3.0
    command:
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
    ports:
      - "80:80"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"

  my-app:
    build: .
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.my-app.rule=Host(`app.localhost`)"
      - "traefik.http.services.my-app.loadbalancer.server.port=8080"
```

### Binary Installation

```bash
# Download latest release
curl -L https://github.com/bawadev/spa-server/releases/latest/download/spa-server-linux-amd64 -o spa-server
chmod +x spa-server

# Serve current directory
./spa-server

# Serve specific directory with compression
./spa-server --directory ./dist --brotli --gzip
```

## Configuration

All options can be set via CLI flags or environment variables:

| Flag | Env Variable | Default | Description |
|------|-------------|---------|-------------|
| `--address, -a` | `ADDRESS` | `0.0.0.0` | Address to bind |
| `--port, -p` | `PORT` | `8080` | Port to listen on |
| `--directory, -d` | `DIRECTORY` | `.` | Directory to serve |
| `--spa` | `SPA_MODE` | `true` | Enable SPA mode |
| `--fallback` | `FALLBACK_FILE` | `index.html` | Fallback file for SPA |
| `--gzip` | `GZIP` | `false` | Enable gzip compression |
| `--brotli` | `BROTLI` | `false` | Enable brotli compression |
| `--threshold` | `THRESHOLD` | `1024` | Min bytes for compression |
| `--cache-max-age` | `CACHE_MAX_AGE` | `604800` | Cache-Control max-age (seconds) |
| `--no-cache-paths` | `NO_CACHE_PATHS` | - | Paths with no-store |
| `--no-compress` | `NO_COMPRESS` | - | Extensions to skip compression |
| `--cache` | `CACHE` | `true` | Enable in-memory cache |
| `--cache-size` | `CACHE_SIZE` | `51200` | LRU cache size (bytes) |
| `--logger` | `LOGGER` | `false` | Enable request logging |
| `--log-pretty` | `LOG_PRETTY` | `false` | Pretty print logs |
| `--health` | `HEALTH_ENDPOINT` | `true` | Enable health endpoints |
| `--security-headers` | `SECURITY_HEADERS` | `true` | Add security headers |
| `--cors` | `CORS` | `false` | Enable CORS |
| `--cors-origin` | `CORS_ORIGIN` | `*` | CORS origin value |

## Examples

### Enable Compression

```bash
# Via CLI
spa-server --brotli --gzip

# Via environment
docker run -e BROTLI=true -e GZIP=true ghcr.io/bawadev/spa-server
```

### Custom Port

```bash
spa-server --port 3000
```

### Disable SPA Mode (Static Site)

```bash
spa-server --spa=false
```

### Enable Logging

```bash
# JSON logs
spa-server --logger

# Pretty logs
spa-server --logger --log-pretty
```

### Kubernetes Health Checks

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Security Headers

When `--security-headers` is enabled (default), these headers are added:

- `X-Frame-Options: SAMEORIGIN`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`

## Comparison with nginx

| Feature | spa-server | nginx |
|---------|-----------|-------|
| Docker image size | ~5 MB | ~140 MB |
| Configuration | Zero config / env vars | Config file required |
| SPA routing | Built-in | Manual config |
| Health endpoints | Built-in | Manual config |
| Brotli compression | Built-in flag | Requires module |
| Security headers | Built-in flag | Manual config |
| Startup time | ~100ms | ~500ms |

## Building from Source

```bash
git clone https://github.com/bawadev/spa-server.git
cd spa-server
go build -o spa-server ./cmd/spa-server
```

## License

MIT License - see [LICENSE](LICENSE)

## Credits

Based on [devforth/spa-to-http](https://github.com/devforth/spa-to-http)
