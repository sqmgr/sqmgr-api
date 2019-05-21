--liquibase formatted sql

-- Copyright 2019 Tom Peters
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--    http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

--changeset weters:1

CREATE TABLE tokens (
	token text NOT NULL PRIMARY KEY
);

CREATE TYPE states AS ENUM ('active', 'pending', 'disabled');

CREATE TABLE users (
	id bigserial NOT NULL PRIMARY KEY,
	email text NULL UNIQUE,
	password_hash text NOT NULL,
	state states NOT NULL DEFAULT 'pending',
	created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
	modified TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE user_confirmations (
	user_id bigint NOT NULL REFERENCES users (id),
	token TEXT NOT NULL UNIQUE,
	created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE pools (
    id bigserial not null primary key,
    token text not null unique references tokens (token),
    user_id bigint not null references users (id),
    name text not null,
    grid_type text not null,
    password_hash text not null,
    locks timestamp,
    created timestamp not null default (now() at time zone 'utc'),
    modified timestamp not null default (now() at time zone 'utc')
);

CREATE TABLE grids (
    id bigserial not null primary key,
    pool_id bigint not null references pools (id),
    ord int not null default 0,
    name text not null,
    home_numbers text[],
    away_numbers text[],
    event_date timestamp,
    created timestamp not null default (now() at time zone 'utc'),
    modified timestamp not null default (now() at time zone 'utc')
);

CREATE INDEX grids_pool_id_ord_idx ON grids(pool_id, ord);

CREATE TABLE grid_settings (
    grid_id bigint not null primary key references grids (id),
    home_team_name text,
    home_team_color_1 text,
    home_team_color_2 text,
    home_team_color_3 text,
    away_team_name text,
    away_team_color_1 text,
    away_team_color_2 text,
    away_team_color_3 text,
    notes text,
    modified timestamp not null default (now() at time zone 'utc')
);

-- determine which pool a user has properly authenticated with
CREATE TABLE pools_users (
    pool_id bigint not null references pools (id),
    user_id bigint not null references users (id),
    created timestamp not null default (now() at time zone 'utc'),
    PRIMARY KEY (user_id, pool_id)
);

CREATE INDEX pools_users_grid_id_idx ON pools_users (pool_id);

CREATE TYPE square_states AS ENUM ('unclaimed', 'claimed', 'paid-partial', 'paid-full');

CREATE TABLE pool_squares (
    id bigserial not null primary key,
    pool_id bigint not null references pools (id),
    square_id int not null default 0,
    state square_states not null default 'unclaimed',
    claimant text,
    user_id bigint references users (id), -- registered users
    session_user_id text, -- non-registered, session-based users
    modified timestamp not null default (now() at time zone 'utc'),
    UNIQUE (pool_id, square_id)
);

CREATE TABLE pool_squares_logs (
    id bigserial not null primary key,
    pool_square_id bigint not null references pool_squares (id),
    user_id bigint references users (id),
    session_user_id text, -- non-registered, session-based users
    state square_states not null default 'unclaimed',
    claimant text,
    remote_addr text,
    note text not null,
    created timestamp not null default (now() at time zone 'utc')
);

CREATE INDEX pool_squares_logs_pool_square_id_idx ON pool_squares_logs (pool_square_id);

--rollback DROP TABLE pool_squares_logs;
--rollback DROP TABLE pool_squares;
--rollback DROP TABLE pools_users;
--rollback DROP TABLE grid_settings;
--rollback DROP TABLE grids;
--rollback DROP TABLE pools;
--rollback DROP TABLE user_confirmations;
--rollback DROP TABLE users;
--rollback DROP TABLE tokens;
--rollback DROP TYPE square_states;
--rollback DROP TYPE states;


--changeset weters:2 splitStatements:false

CREATE FUNCTION new_user(_email text, _password_hash text) RETURNS users
	LANGUAGE plpgsql
	AS $$
DECLARE
	_record users;
BEGIN
	LOCK TABLE users IN SHARE UPDATE EXCLUSIVE MODE;
	_record.id = -1;

	PERFORM 1 FROM users WHERE email = _email;
	IF FOUND THEN
		RETURN _record;
	END IF;

	INSERT INTO users (email, password_hash)
	VALUES (_email, _password_hash)
	RETURNING * INTO _record;

	RETURN _record;
END;
$$;

CREATE FUNCTION set_user_confirmation(_user_id bigint, _token text) RETURNS boolean
	LANGUAGE plpgsql
	AS $$
BEGIN
	PERFORM 1
	FROM user_confirmations
	WHERE user_id = _user_id;

	IF FOUND THEN
		UPDATE user_confirmations
		SET token = _token,
			created = (NOW() AT TIME ZONE 'UTC')
		WHERE user_id = _user_id;
		RETURN true;
	END IF;

	INSERT INTO user_confirmations(user_id, token) VALUES (_user_id, _token);
	RETURN true;
END;
$$;

CREATE FUNCTION new_token(_token text) RETURNS boolean
	LANGUAGE plpgsql
	AS $$
BEGIN
	LOCK TABLE tokens IN SHARE UPDATE EXCLUSIVE MODE;
	PERFORM 1 FROM tokens WHERE token = _token;
	IF FOUND THEN
		RETURN FALSE;
	END IF;

	INSERT INTO tokens (token) VALUES (_token);
	RETURN TRUE;
END;
$$;

CREATE FUNCTION update_pool_square(_id bigint, _state square_states, _claimant text, _user_id bigint, _session_user_id text, _remote_addr text, _note text, _is_admin boolean) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
DECLARE
    _row pool_squares;
    _initial_claim boolean;
    _same_user boolean;
    _user_unclaim boolean;
BEGIN
    SELECT INTO _row * FROM pool_squares WHERE id = _id FOR SHARE;

    _initial_claim := _row.claimant IS NULL AND _row.state = 'unclaimed';
    _same_user := coalesce(_row.user_id, 0) = coalesce(_user_id, 0) AND coalesce(_row.session_user_id, '') = coalesce(_session_user_id, '');
    _user_unclaim := _same_user AND _row.state = 'claimed' AND _state = 'unclaimed';

    IF NOT _is_admin
        AND NOT _initial_claim
        AND NOT _user_unclaim
    THEN
        RETURN FALSE;
    END IF;

    IF _state = 'unclaimed' THEN
        _claimant := NULL;
        _user_id := NULL;
        _session_user_id := NULL;
    END IF;

    UPDATE pool_squares
    SET state = _state,
        claimant = _claimant,
        user_id = _user_id,
        session_user_id = _session_user_id,
        modified = (now() at time zone 'utc')
    WHERE id = _id;

    INSERT INTO pool_squares_logs (pool_square_id, user_id, session_user_id, state, claimant, note, remote_addr) VALUES
    (_id, _user_id, _session_user_id, _state, _claimant, _note, _remote_addr);

    RETURN TRUE;
END;
$$;

CREATE FUNCTION new_grid(_pool_id bigint, _name text) RETURNS grids
    LANGUAGE plpgsql
    AS $$
    declare
        _row grids;
    begin
        INSERT INTO grids (pool_id, name)
        VALUES (_pool_id, _name)
        RETURNING * INTO _row;

        INSERT INTO grid_settings (grid_id)
        VALUES (_row.id);

        RETURN _row;
    end;
$$;

CREATE FUNCTION new_pool(_token text, _user_id bigint, _name text, _grid_type text, _password_hash text, _squares int) RETURNS pools
    LANGUAGE plpgsql
AS $$
DECLARE
    _row pools;
    _counter integer := 0;
BEGIN
    INSERT INTO pools (token, user_id, name, grid_type, password_hash)
    VALUES (_token, _user_id, _name, _grid_type, _password_hash)
           RETURNING * INTO _row;

    LOOP
        EXIT WHEN _counter = _squares;

        -- +1 because the square IDs are 1-based not 0-based
        INSERT INTO pool_squares (pool_id, square_id) VALUES
        (_row.id, _counter + 1);

        _counter := _counter + 1;
    END LOOP;

    PERFORM new_grid(_row.id, _name);

    RETURN _row;
END;
$$;

--rollback DROP FUNCTION new_pool(text, bigint, text, text, text, int);
--rollback DROP FUNCTION new_grid(bigint, text);
--rollback DROP FUNCTION update_pool_square(bigint, square_states, text, bigint, text, text, text, boolean);
--rollback DROP FUNCTION new_user(text, text);
--rollback DROP FUNCTION set_user_confirmation(bigint, text);
--rollback DROP FUNCTION new_token(text);
