--liquibase formatted sql

--changeset weters:1

CREATE TABLE square_types (
	key text NOT NULL PRIMARY KEY,
	description text NOT NULL DEFAULT ''
);

INSERT INTO square_types (key, description) VALUES
	('standard-100', 'Standard, 100 Squares'),
	('standard-25', 'Standard, 25 Squares')
;

CREATE TABLE squares (
	token text NOT NULL PRIMARY KEY,
	square_type text NOT NULL references square_types (key),
	admin_password_hash text NOT NULL,
	join_password_hash text,
	squares_unlock TIMESTAMP NOT NULL DEFAULT (NOW() at time zone 'UTC'),
	squares_lock TIMESTAMP,
	created TIMESTAMP NOT NULL DEFAULT (NOW() at time zone 'UTC'),
	modified TIMESTAMP NOT NULL DEFAULT (NOW() at time zone 'UTC')
);

--rollback DROP TABLE squares;
--rollback DROP TABLE square_types;
