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

CREATE FUNCTION delete_grid(_id bigint) RETURNS boolean
    LANGUAGE plpgsql
AS
$$
declare
    _pool_id      bigint;
    _active_grids bigint;
begin
    SELECT INTO _pool_id pool_id
    FROM grids
    WHERE id = _id;

    if not found then
        raise exception 'grid not found by id %', _id;
    end if;

    perform from grids where pool_id = _pool_id for update;

    select into _active_grids count(*)
    from grids
    where pool_id = _pool_id
      and state = 'active';

    if _active_grids <= 1 then
        return false;
    end if;

    update grids set state = 'deleted', modified = (now() at time zone 'utc') where id = _id;
    return true;
end;
$$;

COMMIT;
