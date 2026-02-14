CREATE TABLE pool_invites (
    token       TEXT PRIMARY KEY,
    pool_id     BIGINT NOT NULL REFERENCES pools(id),
    check_id    INTEGER NOT NULL DEFAULT 0,
    expires_at  TIMESTAMP NOT NULL,
    created     TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);
CREATE INDEX pool_invites_pool_id_idx ON pool_invites(pool_id);
