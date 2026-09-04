# sqmgr-api - The backend for SqMGR

[![CI/CD](https://github.com/sqmgr/sqmgr-api/actions/workflows/main.yaml/badge.svg?branch=master)](https://github.com/sqmgr/sqmgr-api/actions/workflows/main.yaml)
[![Latest tag](https://img.shields.io/github/v/tag/sqmgr/sqmgr-api?label=version)](https://github.com/sqmgr/sqmgr-api/tags)
[![Go version](https://img.shields.io/github/go-mod/go-version/sqmgr/sqmgr-api)](go.mod)
[![License: AGPL v3](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)

SqMGR is a web application for managing squares pools. This repository contains the Go backend API that powers [sqmgr.com](https://sqmgr.com).

## Requirements

- [Go](https://golang.org/doc/install) 1.25+
- [Docker](https://docs.docker.com/install/) (for local PostgreSQL)

## Getting Started

```bash
make git-hooks   # install git pre-commit hooks
make run         # start the development server
```

Verify you get a response by querying [localhost:8000](http://localhost:8000).

## Project Structure

```
sqmgr-api/
├── cmd/
│   ├── sqmgr-api/                 # Main API server
│   ├── sqmgr-guest-user-cleanup/  # Guest user cleanup utility
│   └── sqmgr-sports-sync/         # ESPN teams/schedule/scores sync
├── internal/
│   ├── config/                    # Configuration management
│   ├── database/                  # Database operations & migrations
│   ├── keylocker/                 # Auth0 JWKS key management
│   ├── server/                    # HTTP server, routing & middleware
│   └── validator/                 # Input validation
├── pkg/
│   ├── auth0/                     # Auth0 Management API client
│   ├── model/                     # Data models & business logic
│   ├── smjwt/                     # JWT utilities
│   ├── sports/                    # Sports data providers
│   └── tokengen/                  # Token generation
├── sql/                           # Database migrations
├── k8s/                           # Kubernetes manifests
└── hack/                          # Development utilities
```

## Makefile Commands

Command | Description
--- | ---
`make run` | Run the web server (generates keys + starts dev database, runs migrations)
`make test` | Run staticcheck, gofmt check, go vet, and unit + integration tests with coverage
`make cover` | Generate and open an HTML code coverage report
`make testdata` | Create test data for the database
`make clean` | Tear down dev environment, remove tools and generated keys
`make format` | Run gofmt on Go code
`make git-hooks` | Install pre-commit hooks
`make dev-db` | Start PostgreSQL Docker container
`make integration-db` | (Re)create the `integration` test database
`make migrations` | Apply database migrations
`make migrations-down` | Rollback migrations (set `ROLLBACK_COUNT`, default `1`)

The database targets honor `PG_HOST` (default `localhost`), `PG_PORT` (default `5432`), and
`PG_DATABASE` (default `postgres`).

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
`auth0_mgmt_domain` | Auth0 Management API domain | _(empty)_
`auth0_mgmt_client_id` | Auth0 Management API client ID | _(empty)_
`auth0_mgmt_client_secret` | Auth0 Management API client secret | _(empty)_
`cors_allowed_origins` | Comma-separated list of allowed CORS origins | `https://sqmgr.com,https://www.sqmgr.com,https://beta.sqmgr.com,http://localhost:8080`

### Command-line Flags

`sqmgr-api`:

Flag | Description | Default
--- | --- | ---
`-addr` | Server listen address | `:8000` (or `ADDR` env var)
`-sql` | Path to SQL migrations directory | `./sql`
`-migrate` | Run database migrations on startup | `false`

`sqmgr-sports-sync`:

Flag | Description | Default
--- | --- | ---
`-sync-teams` | Sync teams from ESPN for all leagues | `false`
`-sync-schedule` | Sync upcoming game schedule | `false`
`-sync-scores` | Sync scores for in-progress/recent games | `false`
`-league` | Limit sync to one league (`nfl`, `nba`, `wnba`, `ncaab`, `ncaaf`) | _(all)_
`-dry-run` | Don't persist changes to the database | `false`

`sqmgr-guest-user-cleanup`:

Flag | Description | Default
--- | --- | ---
`-dry-run` | Only output what would be deleted | `false`

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
`GET` | `/pool/{token}/squares/public` | Get the public (read-only) view of a pool's squares
`GET` | `/pool/{token}/events` | Server-sent event stream of pool/score updates
`POST` | `/user/guest` | Create a guest user account
`GET` | `/sports/leagues` | List supported leagues
`GET` | `/sports/events` | List sporting events
`GET` | `/sports/events/{id}` | Get a single sporting event
`GET` | `/sports/teams` | List teams

The `/bdl/*` paths are deprecated aliases for the corresponding `/sports/*` paths and are
retained for backwards compatibility.

### Authenticated Endpoints

Method | Path | Description
--- | --- | ---
`GET` | `/user/self` | Get current user info
`GET` | `/user/self/stats` | Get current user stats
`POST` | `/pool` | Create a new pool
`GET` | `/pool/{token}` | Get pool details
`POST` | `/pool/{token}` | Update pool settings _(manager)_
`POST` | `/pool/{token}/member` | Add member to pool
`GET` | `/pool/{token}/grid` | List grids in pool
`GET` | `/pool/{token}/grid/{id}` | Get specific grid
`POST` | `/pool/{token}/grid/{id}` | Update grid _(manager)_
`DELETE` | `/pool/{token}/grid/{id}` | Delete grid
`POST` | `/pool/{token}/grid/{id}/square/{square_id}/annotation` | Add a square annotation
`DELETE` | `/pool/{token}/grid/{id}/square/{square_id}/annotation` | Remove a square annotation
`GET` | `/pool/{token}/square` | List squares
`GET` | `/pool/{token}/square/{id}` | Get square details
`POST` | `/pool/{token}/square/{id}` | Update square (claim/unclaim)
`POST` | `/pool/{token}/squares/bulk` | Bulk update squares _(manager)_
`GET` | `/pool/{token}/invitetoken` | Get invite token _(manager)_
`GET` | `/pool/{token}/log` | Get activity log _(manager)_
`GET` | `/pool/{token}/members/emails` | List pool member email addresses _(manager)_
`GET` | `/user/{id}/pool/{membership}` | Get user pools (membership: own/belong)
`DELETE` | `/user/{id}/pool/{token}` | Leave or remove pool
`POST` | `/user/{id}/guestjwt` | Issue a guest JWT for the user

### Admin Endpoints

Require site admin privileges.

Method | Path | Description
--- | --- | ---
`GET` | `/admin/stats` | Site-wide stats
`GET` | `/admin/pools` | List pools
`GET` | `/admin/users` | List users
`GET` | `/admin/user/{id}` | Get a user
`GET` | `/admin/user/{id}/pools` | List a user's pools
`POST` | `/admin/pool/{token}/join` | Join a pool as an admin
`GET` | `/admin/events` | List sporting events
`GET` | `/admin/events/{id}/grids` | List grids tied to an event

## Authentication

The API supports two JWT issuers:

1. **Auth0** - For authenticated users via OAuth/OIDC
2. **SqMGR** - For guest user sessions

All authenticated requests require a valid JWT in the `Authorization: Bearer <token>` header with audience `api.sqmgr.com`.

## Rate Limiting

- 10 requests/second per IP with burst of 20 on all routes
- 5 failed authentication attempts per minute per IP on authenticated routes
- Respects `X-Forwarded-For` and `X-Real-IP` headers

## Database

PostgreSQL with migrations managed via [golang-migrate](https://github.com/golang-migrate/migrate).
Local development uses `postgres:11`; CI runs against `postgres:16`.

Run migrations manually:
```bash
make migrations
```

```bash
make migrations-down ROLLBACK_COUNT=2
```

## Docker

Build the Docker image:
```bash
docker build --build-arg VERSION=1.0.0 -t sqmgr-api .
```

The image exposes port 8000 and includes the `sqmgr-api`, `sqmgr-guest-user-cleanup`, and
`sqmgr-sports-sync` binaries.

## CI/CD

[`.github/workflows/main.yaml`](.github/workflows/main.yaml) runs three jobs:

- **test** — migrations, staticcheck, gofmt check, `go vet`, and `go test -race` with coverage
  against a PostgreSQL service container. Runs on pull requests and pushes to `master`.
- **build** — builds and pushes the image to `ghcr.io` on pushes to `master` and `v*` tags.
- **deploy** — rolls the image out to Kubernetes on `v*` tags or manual dispatch.

## Deployment

Kubernetes manifests are provided in the `k8s/` directory:
- `deployment.yaml` - Main API deployment
- `service.yaml` - Service configuration
- `cronjob.yaml` - Guest user cleanup scheduled job
- `sports-sync-cronjob.yaml` - Teams (weekly), schedule (every 8h), and score (every 5m) sync jobs
- `local.yaml` - Config/JWT key secrets for running against a local cluster

## License

GNU Affero General Public License v3.0 - See [LICENSE](LICENSE) for details.
