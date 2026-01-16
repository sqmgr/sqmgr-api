# sqmgr-api - The backend for SqMGR

[![Test](https://github.com/sqmgr/sqmgr-api/workflows/Test/badge.svg)](https://github.com/sqmgr/sqmgr-api/actions?query=workflow%3ATest)
[![Build](https://github.com/sqmgr/sqmgr-api/workflows/Build/badge.svg)](https://github.com/sqmgr/sqmgr-api/actions?query=workflow%3ABuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/sqmgr/sqmgr-api)](https://goreportcard.com/report/github.com/sqmgr/sqmgr-api)

SqMGR is a web application for managing football squares pools. This repository contains the Go backend API that powers [sqmgr.com](https://sqmgr.com).

## Requirements

- [Go](https://golang.org/doc/install) 1.24+
- [Docker](https://docs.docker.com/install/) (for local PostgreSQL)

## Getting Started

```bash
make git-hooks   # install git pre-commit hooks
make run         # start the development server
```

Verify you get a response by querying [localhost:5000](http://localhost:5000).

## Project Structure

```
sqmgr-api/
├── cmd/
│   ├── sqmgr-api/                 # Main API server
│   └── sqmgr-guest-user-cleanup/  # Guest user cleanup utility
├── internal/
│   ├── config/                    # Configuration management
│   ├── database/                  # Database operations & migrations
│   ├── keylocker/                 # Auth0 JWKS key management
│   ├── server/                    # HTTP server & routing
│   └── validator/                 # Input validation
├── pkg/
│   ├── model/                     # Data models & business logic
│   ├── smjwt/                     # JWT utilities
│   └── tokengen/                  # Token generation
├── sql/                           # Database migrations
├── k8s/                           # Kubernetes manifests
└── hack/                          # Development utilities
```

## Makefile Commands

Command | Description
--- | ---
`make run` | Run the web server (generates keys + starts dev database)
`make test` | Run unit and integration tests with coverage
`make cover` | Generate HTML code coverage report
`make testdata` | Create test data for the database
`make clean` | Tear down dev environment and remove tools
`make format` | Run gofmt on Go code
`make git-hooks` | Install pre-commit hooks
`make dev-db` | Start PostgreSQL Docker container
`make migrations` | Apply database migrations
`make migrations-down` | Rollback migrations

## Configuration

Configuration is read from (in order of precedence):

1. Environment variables with `SQMGR_CONF_` prefix (e.g., `SQMGR_CONF_JWT_PUBLIC_KEY`)
2. Config file in current directory (`./config.yaml` or `./config.json`)
3. Config file in `/etc/sqmgr/` (`/etc/sqmgr/config.yaml` or `/etc/sqmgr/config.json`)

### Configuration Options

Key | Description | Default
--- | --- | ---
`dsn` | PostgreSQL connection string | `host=localhost port=5432 user=postgres sslmode=disable`
`jwt_private_key` | Path to PEM private key | **Required**
`jwt_public_key` | Path to PEM public key | **Required**
`auth0_jwks_url` | Auth0 JWKS endpoint | `https://sqmgr.auth0.com/.well-known/jwks.json`

### Command-line Flags

Flag | Description | Default
--- | --- | ---
`-addr` | Server listen address | `:5000` (or `ADDR` env var)
`-sql` | Path to SQL migrations directory | `./sql`
`-migrate` | Run database migrations on startup | `false`

### Environment Variables

Variable | Description
--- | ---
`ADDR` | Server listen address
`LOG_LEVEL` | Logging level (debug, info, warn, error)
`SQMGR_VERSION` | Application version (shown in health endpoint)

## API Endpoints

### Public Endpoints

Method | Path | Description
--- | --- | ---
`GET` | `/` | Health check (returns status and version)
`GET` | `/pool/configuration` | Get pool configuration options
`POST` | `/user/guest` | Create a guest user account

### Authenticated Endpoints

Method | Path | Description
--- | --- | ---
`GET` | `/user/self` | Get current user info
`POST` | `/pool` | Create a new pool
`GET` | `/pool/{token}` | Get pool details
`POST` | `/pool/{token}` | Update pool settings
`POST` | `/pool/{token}/member` | Add member to pool
`GET` | `/pool/{token}/grid` | List grids in pool
`GET` | `/pool/{token}/grid/{id}` | Get specific grid
`POST` | `/pool/{token}/grid/{id}` | Update grid
`DELETE` | `/pool/{token}/grid/{id}` | Delete grid
`GET` | `/pool/{token}/square` | List squares
`GET` | `/pool/{token}/square/{id}` | Get square details
`POST` | `/pool/{token}/square/{id}` | Update square (claim/unclaim)
`GET` | `/pool/{token}/invitetoken` | Get invite token
`GET` | `/pool/{token}/log` | Get activity log
`GET` | `/user/{id}/pool/{membership}` | Get user pools (membership: own/belong)
`DELETE` | `/user/{id}/pool/{token}` | Leave or remove pool

## Authentication

The API supports two JWT issuers:

1. **Auth0** - For authenticated users via OAuth/OIDC
2. **SqMGR** - For guest user sessions

All authenticated requests require a valid JWT in the `Authorization: Bearer <token>` header with audience `api.sqmgr.com`.

## Rate Limiting

- 10 requests/second per IP with burst of 20
- Respects `X-Forwarded-For` and `X-Real-IP` headers

## Database

PostgreSQL 11+ with migrations managed via [golang-migrate](https://github.com/golang-migrate/migrate).

Run migrations manually:
```bash
make migrations        # apply all pending migrations
make migrations-down   # rollback (set MIGRATION_DOWN_COUNT for multiple)
```

## Docker

Build the Docker image:
```bash
docker build --build-arg SQMGR_VERSION=1.0.0 -t sqmgr-api .
```

The image exposes port 5000 and includes both `sqmgr-api` and `sqmgr-guest-user-cleanup` binaries.

## Deployment

Kubernetes manifests are provided in the `k8s/` directory:
- `deployment.yaml` - Main API deployment
- `service.yaml` - Service configuration
- `cronjob.yaml` - Guest user cleanup scheduled job

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
