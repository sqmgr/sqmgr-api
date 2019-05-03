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

CREATE TABLE states (
	name text NOT NULL PRIMARY KEY
);

INSERT INTO states (name) VALUES
	('active'),
	('pending'),
	('disabled');

CREATE TABLE users (
	id bigserial NOT NULL PRIMARY KEY,
	email text NULL UNIQUE,
	password_hash text NOT NULL,
	state text NOT NULL DEFAULT 'pending' REFERENCES states (name),
	created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
	modified TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE user_confirmations (
	user_id bigint NOT NULL REFERENCES users (id),
	token TEXT NOT NULL UNIQUE,
	created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE grids (
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

-- determine which grid a user has properly authenticated with
CREATE TABLE grids_users (
    grid_id bigint not null references grids (id),
    user_id bigint not null references users (id),
    created timestamp not null default (now() at time zone 'utc'),
    PRIMARY KEY (user_id, grid_id)
);

CREATE INDEX grids_users_grid_id_idx ON grids_users (grid_id);


CREATE TABLE grid_square_states (
    state text not null primary key
);

INSERT INTO grid_square_states (state) VALUES
('unclaimed'),
('claimed'),
('paid-partial'),
('paid-full');

CREATE TABLE grid_squares (
    id bigserial not null primary key,
    grid_id bigint not null references grids (id),
    square_id int not null default 0,
    state text not null default 'unclaimed' references grid_square_states (state),
    claimant text,
    modified timestamp not null default (now() at time zone 'utc'),
    UNIQUE (grid_id, square_id)
);

CREATE TABLE grid_squares_logs (
    id bigserial not null primary key,
    grid_square_id bigint not null references grid_squares (id),
    user_id bigint references users (id),
    state text not null default 'unclaimed' references grid_square_states (state),
    claimant text,
    remote_addr text,
    note text not null,
    created timestamp not null default (now() at time zone 'utc')
);

CREATE INDEX grid_squares_logs_grid_square_id_idx ON grid_squares_logs (grid_square_id);

--rollback DROP TABLE grid_squares_logs;
--rollback DROP TABLE grid_squares;
--rollback DROP TABLE grid_square_states;
--rollback DROP TABLE grids_users;
--rollback DROP TABLE grid_settings;
--rollback DROP TABLE grids;
--rollback DROP TABLE user_confirmations;
--rollback DROP TABLE users;
--rollback DROP TABLE states;
--rollback DROP TABLE tokens;


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

CREATE FUNCTION new_grid(_token text, _user_id bigint, _name text, _grid_type text, _password_hash text, _squares int) RETURNS grids
	LANGUAGE plpgsql
	AS $$
DECLARE
	_row grids;
    _counter integer := 0;
BEGIN
	INSERT INTO grids (token, user_id, name, grid_type, password_hash)
	VALUES (_token, _user_id, _name, _grid_type, _password_hash)
	RETURNING * INTO _row;

	INSERT INTO grid_settings (grid_id)
	VALUES (_row.id);

    LOOP
       EXIT WHEN _counter = _squares;

       INSERT INTO grid_squares (grid_id, square_id) VALUES
       (_row.id, _counter);

       _counter := _counter + 1;
    END LOOP;

	RETURN _row;
END;
$$;

CREATE FUNCTION update_grid_square(_id bigint, _state text, _claimant text, _user_id bigint, _remote_addr text, _note text, _is_admin boolean) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
DECLARE
    _row grid_squares;
BEGIN
    SELECT INTO _row * FROM grid_squares WHERE id = _id FOR SHARE;

    IF NOT _is_admin AND
       (_row.claimant IS NOT NULL OR _row.state != 'unclaimed')
    THEN
        RETURN FALSE;
    END IF;

    UPDATE grid_squares SET state = _state, claimant = _claimant, modified = (now() at time zone 'utc') WHERE id = _id;

    INSERT INTO grid_squares_logs (grid_square_id, user_id, state, claimant, note, remote_addr) VALUES
    (_id, _user_id, _state, _claimant, _note, _remote_addr);

    RETURN TRUE;
END;
$$;

--rollback DROP FUNCTION update_grid_square(bigint, text, text, bigint, text, text, boolean);
--rollback DROP FUNCTION new_user(text, text);
--rollback DROP FUNCTION set_user_confirmation(bigint, text);
--rollback DROP FUNCTION new_token(text);
--rollback DROP FUNCTION new_grid(text, bigint, text, text, text, int);
