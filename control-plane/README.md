# POPSigner Control Plane API

Point-of-Presence signing infrastructure - Control Plane API for managing cryptographic key operations, authentication, billing, and tenant isolation.

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make (optional but recommended)

### Local Development

1. **Start infrastructure services:**

```bash
make docker-up
# or
docker compose -f docker/docker-compose.yml up -d
```

2. **Run the API server:**

```bash
make run
# or
go run ./cmd/server
```

3. **Verify it's working:**

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl http://localhost:8080/ready
# {"status":"ok","database":"connected","redis":"connected"}
```

## Project Structure

```
control-plane/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── database/
│   │   ├── postgres.go          # PostgreSQL connection
│   │   ├── redis.go             # Redis connection
│   │   └── migrations/          # SQL migrations
│   ├── middleware/
│   │   ├── auth.go              # Authentication
│   │   ├── cors.go              # CORS handling
│   │   ├── logging.go           # Request logging
│   │   └── ratelimit.go         # Rate limiting
│   ├── models/                  # Data models
│   ├── pkg/
│   │   ├── errors/              # API error types
│   │   ├── response/            # JSON response helpers
│   │   └── ulid/                # ID generation
│   ├── repository/              # Data access layer
│   ├── service/                 # Business logic
│   └── handler/                 # HTTP handlers
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── config.yaml                  # Sample configuration
├── go.mod
├── go.sum
└── Makefile
```

## Configuration

Configuration can be provided via:

1. **YAML file** (`config.yaml` in current dir or `/etc/popsigner/`)
2. **Environment variables** (prefix: `POPSIGNER_`)

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `POPSIGNER_SERVER_PORT` | HTTP port | 8080 |
| `POPSIGNER_SERVER_ENVIRONMENT` | dev/staging/prod | dev |
| `POPSIGNER_DATABASE_HOST` | PostgreSQL host | localhost |
| `POPSIGNER_DATABASE_PORT` | PostgreSQL port | 5432 |
| `POPSIGNER_DATABASE_USER` | PostgreSQL user | popsigner |
| `POPSIGNER_DATABASE_PASSWORD` | PostgreSQL password | popsigner |
| `POPSIGNER_DATABASE_DATABASE` | PostgreSQL database | popsigner |
| `POPSIGNER_REDIS_HOST` | Redis host | localhost |
| `POPSIGNER_REDIS_PORT` | Redis port | 6379 |
| `POPSIGNER_AUTH_JWT_SECRET` | JWT signing secret | - |

## API Endpoints

### Health Checks

- `GET /health` - Basic health check
- `GET /ready` - Readiness check (includes DB/Redis)

### API v1 (Authenticated)

- `GET /v1/` - API info

*Additional routes will be implemented by other agents.*

## Development

### Run tests

```bash
make test
```

### Run with hot reload

```bash
# Install air first: go install github.com/cosmtrek/air@latest
make dev
```

### Build Docker image

```bash
make docker-build
```

## Database Migrations

Migrations run automatically on server start. To run manually:

```bash
make migrate-up
make migrate-down
```

## License

MIT
