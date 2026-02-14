-- Copyright (C) 2020 Tom Peters
--
-- This program is free software: you can redistribute it and/or modify
-- it under the terms of the GNU Affero General Public License as published by
-- the Free Software Foundation, either version 3 of the License, or
-- (at your option) any later version.
--
-- This program is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
-- GNU Affero General Public License for more details.
--
-- You should have received a copy of the GNU Affero General Public License
-- along with this program.  If not, see <https://www.gnu.org/licenses/>.


BEGIN;

CREATE TABLE tokens
(
    token text NOT NULL PRIMARY KEY
);

CREATE TYPE states AS ENUM ('active', 'pending', 'deleted', 'disabled');

CREATE TYPE stores AS ENUM ('sqmgr', 'auth0');

CREATE TABLE users
(
    id       bigserial not null primary key,
    store    stores    not null,
    store_id text      not null,
    created  timestamp not null default (now() at time zone 'utc'),
    UNIQUE (store, store_id)
);

CREATE TABLE pools
(
    id            bigserial not null primary key,
    token         text      not null unique references tokens (token),
    user_id       bigint    not null references users (id),
    name          text      not null,
    grid_type     text      not null,
    password_hash text      not null,
    locks         timestamp,
    created       timestamp not null default (now() at time zone 'utc'),
    modified      timestamp not null default (now() at time zone 'utc')
);

CREATE TABLE grids
(
    id             bigserial not null primary key,
    pool_id        bigint    not null references pools (id),
    ord            int       not null default 0,
    home_team_name text      null,
    home_numbers   text[],
    away_team_name text      null,
    away_numbers   text[],
    event_date     timestamp,
    state          states    not null default 'active',
    created        timestamp not null default (now() at time zone 'utc'),
    modified       timestamp not null default (now() at time zone 'utc')
);

CREATE INDEX grids_pool_id_ord_idx ON grids (pool_id, ord);

CREATE TABLE grid_settings
(
    grid_id           bigint    not null primary key references grids (id),
    home_team_color_1 text,
    home_team_color_2 text,
    home_team_color_3 text,
    away_team_color_1 text,
    away_team_color_2 text,
    away_team_color_3 text,
    notes             text,
    modified          timestamp not null default (now() at time zone 'utc')
);

-- determine which pool a user has properly authenticated with
CREATE TABLE pools_users
(
    pool_id bigint    not null references pools (id),
    user_id bigint    not null references users (id),
    created timestamp not null default (now() at time zone 'utc'),
    PRIMARY KEY (user_id, pool_id)
);

CREATE INDEX pools_users_grid_id_idx ON pools_users (pool_id);

CREATE TYPE square_states AS ENUM ('unclaimed', 'claimed', 'paid-partial', 'paid-full');

CREATE TABLE pool_squares
(
    id              bigserial     not null primary key,
    pool_id         bigint        not null references pools (id),
    square_id       int           not null default 0,
    state           square_states not null default 'unclaimed',
    claimant        text,
    user_id         bigint references users (id), -- registered users
    modified        timestamp     not null default (now() at time zone 'utc'),
    UNIQUE (pool_id, square_id)
);

CREATE TABLE pool_squares_logs
(
    id              bigserial     not null primary key,
    pool_square_id  bigint        not null references pool_squares (id),
    user_id         bigint references users (id),
    state           square_states not null default 'unclaimed',
    claimant        text,
    remote_addr     text,
    note            text          not null,
    created         timestamp     not null default (now() at time zone 'utc')
);

CREATE INDEX pool_squares_logs_pool_square_id_idx ON pool_squares_logs (pool_square_id);

COMMIT;
