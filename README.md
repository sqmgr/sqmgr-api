# SqMGR - An online football squares manager application

## To start development

Get a working [Go](https://golang.org/doc/install) and [Docker](https://docs.docker.com/install/) environment setup. Then you can start the development server using the following:

```
$ make git-hooks   # install any necessary git-hooks
$ make dev-db      # starts a local PostgreSQL database in docker
$ make migrations  # runs db migrations
$ make web         # builds static assets
$ make run         # run the web server
```

Open your browser and navigate to [localhost:8080](http://localhost:8080).

## Configuration

Configuration is specified one of three ways:

1. Environment variable with a `SQMGR_CONF_` prefix, e.g., `SQMGR_CONF_JWT_PUBLIC_KEY`
2. In a file in the current directory named like `./config.yaml` or `./config.json`
3. In a file in the `/etc/sqmgr/` directory like `/etc/sqmgr/config.yaml` or `/etc/sqmgr/config.json`

The following options can be set:

Key | Description
--- | ---
`dsn` | Database DSN. Default is `host=localhost port=5432 user=postgres sslmode=disable`
`from_address` | Email to use as from address. Default is `weters19@gmail.com`
`jwt_private_key` | Required. Path to a PEM private key
`jwt_public_key` | Required. Path to a PEM public key
`opaque_salt` | Used as the salt when hashing user IDs
`recaptcha_enabled` | Is reCAPTCHA v3 enabled? Default is `true`
`recaptcha_secret_key` | Google reCAPTCHA v3 secret key. Required if `recaptcha_enabled=true`
`recaptcha_site_key` | Google reCAPTCHA v3 site key. Required if `recaptcha_enabled=true`
`session_auth_key` | Key used for authenticating sessions. Should be 64 bytes
`session_enc_key ` | Key used for encrypting sessions. Should be 32 bytes
`smtp` | Address of an SMTP server
`url` | Public address of the server

