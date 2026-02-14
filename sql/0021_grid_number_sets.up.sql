-- Copyright (C) 2024 Tom Peters
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

-- Enum for number set configurations
CREATE TYPE number_set_config AS ENUM (
    'standard', -- Standard (one set for all)
    'q1234',     -- 1st, 2nd, 3rd, 4th quarter
    'q123f',     -- 1st, 2nd, 3rd, Final
    'hf',       -- Half, Final
    'h4'        -- Half, 4th
);

-- Enum for individual number set identifiers
CREATE TYPE number_set_type AS ENUM (
    'all', 'q1', 'q2', 'q3', 'q4', 'half', 'final'
);

-- Add configuration column to pools table (pool-level config)
ALTER TABLE pools
    ADD COLUMN number_set_config number_set_config NOT NULL DEFAULT 'standard';

-- New table for multiple number sets per grid
CREATE TABLE grid_number_sets
(
    id           bigserial       NOT NULL PRIMARY KEY,
    grid_id      bigint          NOT NULL REFERENCES grids (id) ON DELETE CASCADE,
    set_type     number_set_type NOT NULL,
    home_numbers integer[],
    away_numbers integer[],
    manual_draw  boolean         NOT NULL DEFAULT false,
    created      timestamp       NOT NULL DEFAULT (now() at time zone 'utc'),
    modified     timestamp       NOT NULL DEFAULT (now() at time zone 'utc'),
    UNIQUE (grid_id, set_type)
);

CREATE INDEX grid_number_sets_grid_id_idx ON grid_number_sets (grid_id);

-- Migrate existing grids with numbers to new table
-- Cast text[] to integer[] since the original grids table stores numbers as text
INSERT INTO grid_number_sets (grid_id, set_type, home_numbers, away_numbers, manual_draw)
SELECT id,
       'all'::number_set_type, ARRAY(SELECT unnest(home_numbers)::integer),
       ARRAY(SELECT unnest(away_numbers)::integer),
       manual_draw
FROM grids
WHERE home_numbers IS NOT NULL;

-- Update the new_pool function to accept number_set_config
DROP FUNCTION IF EXISTS new_pool(_token text, _user_id bigint, _name text, _grid_type text, _password_hash text, _squares int);

CREATE FUNCTION new_pool(_token text, _user_id bigint, _name text, _grid_type text, _password_hash text,
                         _squares int, _number_set_config number_set_config DEFAULT 'standard') RETURNS pools
    LANGUAGE plpgsql
AS
$$
DECLARE
_row     pools;
    _counter
integer := 0;
BEGIN
INSERT INTO pools (token, user_id, name, grid_type, password_hash, number_set_config)
VALUES (_token, _user_id, _name, _grid_type, _password_hash, _number_set_config) RETURNING *
INTO _row;

LOOP
EXIT WHEN _counter = _squares;

        -- +1 because the square IDs are 1-based not 0-based
INSERT INTO pool_squares (pool_id, square_id)
VALUES (_row.id, _counter + 1);

_counter
:= _counter + 1;
END LOOP;

    PERFORM
new_grid(_row.id, 999999);

RETURN _row;
END;
$$;

COMMIT;
