--liquibase formatted sql

--changeset weters:1

CREATE TABLE square_types (
	key text NOT NULL PRIMARY KEY,
	description text NOT NULL DEFAULT '',
	ord int NOT NULL DEFAULT 0
);

CREATE INDEX square_types_ord_idx ON square_types (ord);

INSERT INTO square_types (key, description, ord) VALUES
	('standard-100', 'Standard, 100 Squares', 0),
	('standard-25', 'Standard, 25 Squares', 1)
;

CREATE TABLE squares (
	token text NOT NULL PRIMARY KEY,
	name text NOT NULL,
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
