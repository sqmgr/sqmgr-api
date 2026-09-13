BEGIN;

-- OAuth 2.1 clients registered dynamically (RFC 7591) so MCP clients such as
-- Claude Desktop can connect to the admin analytics server.
CREATE TABLE oauth_clients
(
    id            TEXT      NOT NULL PRIMARY KEY,
    name          TEXT      NOT NULL,
    redirect_uris TEXT[]    NOT NULL,
    created       TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);

-- Short-lived, single-use authorization codes bound to a PKCE challenge.
CREATE TABLE oauth_authorization_codes
(
    code_hash      TEXT      NOT NULL PRIMARY KEY,
    client_id      TEXT      NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    user_id        BIGINT    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    redirect_uri   TEXT      NOT NULL,
    code_challenge TEXT      NOT NULL,
    scope          TEXT      NOT NULL,
    expires_at     TIMESTAMP NOT NULL,
    created        TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);

-- Refresh tokens, rotated on every use.
CREATE TABLE oauth_refresh_tokens
(
    token_hash TEXT      NOT NULL PRIMARY KEY,
    client_id  TEXT      NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    user_id    BIGINT    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    scope      TEXT      NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created    TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);

CREATE INDEX oauth_refresh_tokens_user_id_idx ON oauth_refresh_tokens (user_id);

COMMIT;
