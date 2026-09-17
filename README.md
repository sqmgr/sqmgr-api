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
│   ├── sqmgr-email-backfill/      # One-time Auth0 email backfill
│   ├── sqmgr-guest-user-cleanup/  # Guest user cleanup utility
│   └── sqmgr-sports-sync/         # ESPN teams/schedule/scores sync (thin wrapper over pkg/sportsync)
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
│   ├── sportsync/                 # ESPN → database sync used by the sync command and admin API
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
`public_url` | Externally reachable base URL of this API (OAuth issuer and MCP resource identifier) | `https://api.sqmgr.com`
`frontend_url` | Base URL of the web app, which hosts the OAuth authorization page | `https://sqmgr.com`

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

`sqmgr-email-backfill` (stores Auth0 emails for users who haven't logged in since emails began being saved at login;
requires the `auth0_mgmt_*` settings and is safe to re-run):

Flag | Description | Default
--- | --- | ---
`-dry-run` | Only output which users would be updated | `false`
`-rate` | Maximum Auth0 Management API requests per second | `2`
`-batch-size` | Number of users to load from the database at a time | `100`

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
`GET` | `/admin/user/{id}/joined-pools` | List pools a user has joined
`GET` | `/admin/pool/{token}` | Pool details: settings, square states, invites, grids
`GET` | `/admin/pool/{token}/members` | Everyone with access to a pool
`GET` | `/admin/pool/{token}/activity` | The pool's square change log
`GET` | `/admin/pool/{token}/squares` | The pool's squares and who claimed them
`POST` | `/admin/pool/{token}/action` | Archive, lock, reset password, transfer ownership, or revoke invites
`POST` | `/admin/pool/{token}/join` | Join a pool as an admin
`GET` | `/admin/analytics/timeseries` | A metric counted per day, week, month, or year
`GET` | `/admin/analytics/fill-rates` | How full pools get
`GET` | `/admin/analytics/breakdown` | Pools, grids, or squares grouped by one dimension
`GET` | `/admin/analytics/engagement` | Engagement summary for a date range
`GET` | `/admin/analytics/top-creators` | Users ranked by pools created
`GET` | `/admin/events` | List sporting events
`GET` | `/admin/events/{id}/grids` | List grids tied to an event
`POST` | `/admin/events/{id}/refresh` | Fetch an event from ESPN right now
`POST` | `/admin/events/{id}/override` | Manually correct an event's status and scores
`DELETE` | `/admin/events/{id}/override` | Let the sync job update the event again
`GET` | `/admin/sports/status` | Last sync run per type, in-progress manual sync, stale events
`GET` | `/admin/sports/sync-runs` | Recent sports sync runs
`POST` | `/admin/sports/sync` | Start a teams, schedule, or scores sync in the background
`GET` | `/admin/audit` | Every write action a site admin has performed
`POST` | `/admin/mcp` | Read-only analytics [MCP](https://modelcontextprotocol.io) server (see below)

Every admin write action is recorded in the `admin_audit_log` table with the admin, target, an
optional reason, and before/after details. Manually overridden events carry a `manual_override`
flag that the sync job respects until the override is cleared.

### Admin Analytics MCP Server

`/admin/mcp` exposes a read-only [Model Context Protocol](https://modelcontextprotocol.io) server over
streamable HTTP so a site admin can ask an MCP client (Claude Desktop, Claude Code, etc.) questions about how the
site is being used. Only site admins can use it.

**Connecting Claude Desktop:** Settings → Connectors → Add custom connector, name it and enter
`https://api.sqmgr.com/admin/mcp`. Claude opens a browser tab at `https://sqmgr.com/oauth/authorize`; log in as
usual, click **Approve**, and the connector is ready. Claude Code works the same way:

```bash
claude mcp add --transport http sqmgr-admin https://api.sqmgr.com/admin/mcp
```

Behind the scenes the API is a small OAuth 2.1 authorization server (see below), so no tokens are copied by hand
and access is refreshed automatically for 30 days per approval. Sending a site admin's regular API JWT as
`Authorization: Bearer <jwt>` also works for scripted use.

The server is stateless (each POST is independent; `GET`/`DELETE` return `405`) and returns plain JSON responses.
Every tool runs inside a PostgreSQL `READ ONLY` transaction with a 30 second statement timeout, so nothing exposed
here can modify data. All tools are annotated `readOnlyHint`. The ad hoc query tool additionally blocks a list of
side-effect functions, but what it can *read* is bounded only by the database role in `dsn`, so that role should
not be a superuser in production.

Tool | Description
--- | ---
`get_site_stats` | Site-wide totals (pools, users, guests, grids, claimed squares, memberships), optionally within a date range
`get_time_series` | Count a metric (`pools_created`, `users_registered`, `guest_users_created`, `squares_claimed`, `grids_created`, `pool_members_joined`) per day/week/month/year, zero-filled for charting
`get_pool_fill_rates` | Share of pools that are fully / partially / never claimed, plus average, median, and a fill-percentage histogram
`get_pool_breakdown` | Group pools (or grids/squares) by grid type, number set config, archived, password required, owner account type, league, or square state
`get_engagement_summary` | New users, distinct/repeat/returning pool creators, claims by registered vs. guest vs. anonymous users, averages per pool
`list_pools` | Search and page pools with owner email, member/grid counts, and fill percentage; filter by date, grid type, archived, fill range
`get_pool` | Full details for one pool: settings, owner, squares by state, active invites, grids and their linked sports events
`list_pool_squares` | Claimed squares in a pool with claimant name and, for registered users, their email address
`list_pool_members` | Owner, managers, and members of a pool with email, join date, and squares claimed
`list_pool_activity` | The pool's square change log (claims, unclaims, payment changes), newest first
`list_users` | Search and page registered, guest, or all users with pools owned/joined and squares claimed
`get_user` | One user by ID or email with counts and the pools they own and belong to
`get_top_pool_creators` | Users ranked by pools created in a date range
`list_popular_events` | Sports events ranked by linked grids, optionally by league and date
`list_sports_sync_runs` | Recent sports sync job runs and errors
`describe_schema` | Tables, columns, and enum types, for writing ad hoc queries
`run_sql_query` | A single ad hoc `SELECT` (wrapped as a subquery, row-capped, `password_hash` and escaped identifiers rejected) for anything the other tools do not cover

Dates accept `YYYY-MM-DD` (UTC calendar days; an end date includes the whole day) or RFC3339 timestamps (end
exclusive). Omit both for all time.

### OAuth Endpoints

These implement the authorization flow MCP clients expect (OAuth 2.1 with PKCE, dynamic client registration, and
discovery metadata). Clients are public; there are no client secrets.

Method | Path | Description
--- | --- | ---
`GET` | `/.well-known/oauth-authorization-server` | Authorization server metadata (RFC 8414)
`GET` | `/.well-known/oauth-protected-resource[/admin/mcp]` | Protected resource metadata for the MCP endpoint (RFC 9728)
`POST` | `/oauth/register` | Dynamic client registration (RFC 7591); `redirect_uris` must be `https` or `http://localhost`
`POST` | `/oauth/token` | Exchange an authorization code (with `code_verifier`) or a refresh token for tokens
`GET` | `/oauth/client/{id}` | Client name and redirect URIs, used by the authorization page _(authenticated)_
`POST` | `/oauth/authorize` | Approve or deny a request; returns the redirect URL carrying the code _(site admin)_

The authorization page itself is `https://sqmgr.com/oauth/authorize` in the web app, which logs the user in with
Auth0, confirms they are a site admin, and calls `POST /oauth/authorize`. Access tokens are JWTs signed with the
SqMGR key with the MCP endpoint URL as their audience and expire after 1 hour; refresh tokens are single-use,
rotate on every refresh, and expire after 30 days. Authorization codes are single-use and expire after 10
minutes. A grant stops working as soon as the user is no longer a site admin.

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

The image exposes port 8000 and includes the `sqmgr-api`, `sqmgr-email-backfill`, `sqmgr-guest-user-cleanup`,
and `sqmgr-sports-sync` binaries.

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
