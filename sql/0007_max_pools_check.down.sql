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

DROP FUNCTION new_grid(bigint, int);
CREATE FUNCTION new_grid(_pool_id bigint) RETURNS grids
    LANGUAGE plpgsql
AS
$$
declare
    _row grids;
begin
    INSERT INTO grids (pool_id, ord)
    VALUES (_pool_id, (SELECT COALESCE(MAX(ord), -1) + 1 FROM grids WHERE pool_id = _pool_id)) RETURNING * INTO _row;

    INSERT INTO grid_settings (grid_id)
    VALUES (_row.id);

    RETURN _row;
end;
$$;

DROP FUNCTION new_pool(text, bigint, text, text, text, int);
CREATE FUNCTION new_pool(_token text, _user_id bigint, _name text, _grid_type text, _password_hash text,
                         _squares int) RETURNS pools
    LANGUAGE plpgsql
AS
$$
DECLARE
    _row     pools;
    _counter integer := 0;
BEGIN
    INSERT INTO pools (token, user_id, name, grid_type, password_hash)
    VALUES (_token, _user_id, _name, _grid_type, _password_hash) RETURNING * INTO _row;

    LOOP
        EXIT WHEN _counter = _squares;

        -- +1 because the square IDs are 1-based not 0-based
        INSERT INTO pool_squares (pool_id, square_id)
        VALUES (_row.id, _counter + 1);

        _counter := _counter + 1;
    END LOOP;

    PERFORM new_grid(_row.id);

    RETURN _row;
END;
$$;

COMMIT;
