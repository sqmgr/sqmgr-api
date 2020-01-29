-- Copyright 2020 Tom Peters
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
