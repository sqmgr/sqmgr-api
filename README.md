# sqmgr-api - The backend for SqMGR

[![Test](https://github.com/sqmgr/sqmgr-api/workflows/Test/badge.svg)](https://github.com/sqmgr/sqmgr-api/actions?query=workflow%3ATest)
[![Build](https://github.com/sqmgr/sqmgr-api/workflows/Build/badge.svg)](https://github.com/sqmgr/sqmgr-api/actions?query=workflow%3ABuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/sqmgr/sqmgr-api)](https://goreportcard.com/report/github.com/sqmgr/sqmgr-api)

## To start development

Get a working [Go](https://golang.org/doc/install) and [Docker](https://docs.docker.com/install/) environment setup. Then you can start the development server using the following:

```
$ make git-hooks   # install any necessary git-hooks
$ make run         # run the web server
```

Verify you get a response by querying [localhost:5000](http://localhost:5000).

### Makefile

Below is a list of the most common Makefile commands you'll want to run.

Command | Description
--- | ---
`make run` | This will run the web server. It will also ensure your keys have been generated and your dev database is running
`make test` | Runs the unit and integration tests
`make cover` | Generates a code coverage report
`make testdata` | Creates test data for your database
`make clean` | Tears down your dev environment. Removes any tools.
`make fmt` | Runs gofmt on your go code

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
