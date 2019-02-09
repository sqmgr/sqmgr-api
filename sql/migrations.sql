CREATE TABLE square_types (
	key string primary key,
	description string
);

INSERT INTO square_types ( square_type ) VALUES
	( '25', 'Twenty-five squares' ),
	( '100', 'One-hundred squares' );

CREATE TABLE squares (
	key text primary key,
	name text,
	square_type text not null references square_types (key),
	admin_password_hash text,
	password_hash text,
	notes text,
	locks_at timestamp without time zone,
	created_at timestamp without time zone
);

CREATE TABLE square_numbers (
	square_key text not null references squares (key),
	square int not null,
	value int not null,
	primary key (square_key, square)
);

CREATE TABLE square_claims (
	id bigint not null primary key,
	square_key text not null references squares (key),
	square int not null,
	approved boolean not null default 'f',
	deleted boolean not null default 'f',
	created timestamp without time zone,
	updated tiemstamp without time zone
);
