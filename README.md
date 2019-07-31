# sqmgr-api - The backend for SqMGR

## To start development

Get a working [Go](https://golang.org/doc/install) and [Docker](https://docs.docker.com/install/) environment setup. Then you can start the development server using the following:

```
$ make git-hooks   # install any necessary git-hooks
$ make dev-db      # starts a local PostgreSQL database in docker
$ make migrations  # runs db migrations
$ make run         # run the web server
```

Verify you get a response by querying [localhost:5000](http://localhost:5000).

## Configuration

Configuration is specified one of three ways:

1. Environment variable with a `SQMGR_CONF_` prefix, e.g., `SQMGR_CONF_JWT_PUBLIC_KEY`
2. In a file in the current directory named like `./config.yaml` or `./config.json`
3. In a file in the `/etc/sqmgr/` directory like `/etc/sqmgr/config.yaml` or `/etc/sqmgr/config.json`

The following options can be set:

Key | Description
--- | ---
`dsn` | Database DSN. Default is `host=localhost port=5432 user=postgres sslmode=disable`
`jwt_private_key` | Required. Path to a PEM private key
`jwt_public_key` | Required. Path to a PEM public key
