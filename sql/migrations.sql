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

--rollback DROP FUNCTION new_user(text, text);

--changeset weters:3 splitStatements:false

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

--rollback DROP FUNCTION set_user_confirmation(bigint, text);

--changeset weters:4

CREATE TABLE squares (
	id bigserial not null primary key,
	token text not null unique references tokens (token),
	user_id bigint not null references users (id),
	name text not null,
	squares_type text not null,
	password_hash text not null,
	locks timestamp,
	created timestamp not null default (now() at time zone 'utc'),
	modified timestamp not null default (now() at time zone 'utc')
);

CREATE TABLE squares_settings (
	squares_id bigint not null primary key references squares (id),
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

--rollback DROP TABLE squares_settings;
--rollback DROP TABLE squares;

--changeset weters:5 splitStatements:false

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

CREATE FUNCTION new_squares(_token text, _user_id bigint, _name text, _squares_type text, _password_hash text) RETURNS squares
	LANGUAGE plpgsql
	AS $$
DECLARE
	_row squares;
BEGIN
	INSERT INTO squares (token, user_id, name, squares_type, password_hash)
	VALUES (_token, _user_id, _name, _squares_type, _password_hash)
	RETURNING * INTO _row;

	INSERT INTO squares_settings (squares_id)
	VALUES (_row.id);

	RETURN _row;
END;
$$;

--rollback DROP FUNCTION new_token(text);
--rollback DROP FUNCTION new_squares(text, bigint, text, text, text);

--changeset weters:6

-- determine which squares a user has properly authenticated with
CREATE TABLE squares_users (
	squares_id bigint not null references squares (id),
	user_id bigint not null references users (id),
	created timestamp not null default (now() at time zone 'utc'),
	PRIMARY KEY (user_id, squares_id)
);

CREATE INDEX squares_users_squares_id_idx ON squares_users (squares_id);

--rollback DROP TABLE squares_users;
