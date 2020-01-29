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

CREATE FUNCTION get_user(_store stores, _store_id text) RETURNS users
    LANGUAGE plpgsql
AS
$$
declare
    _record users;
begin
    SELECT *
    INTO _record
    FROM users
    WHERE store = _store
      AND store_id = _store_id;

    if found then
        return _record;
    end if;

    insert into users (store, store_id)
    values (_store, _store_id) returning * into _record;

    return _record;
end;
$$;


CREATE FUNCTION new_token(_token text) RETURNS boolean
    LANGUAGE plpgsql
AS
$$
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

CREATE FUNCTION update_pool_square(_id bigint, _state square_states, _claimant text, _user_id bigint,
                                   _remote_addr text, _note text, _is_admin boolean) RETURNS boolean
    LANGUAGE plpgsql
AS
$$
DECLARE
    _row           pool_squares;
    _initial_claim boolean;
    _same_user     boolean;
    _user_unclaim  boolean;
BEGIN
    SELECT INTO _row * FROM pool_squares WHERE id = _id FOR SHARE;

    _initial_claim := _row.claimant IS NULL AND _row.state = 'unclaimed';
    _same_user := coalesce(_row.user_id, 0) = coalesce(_user_id, 0);
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
    END IF;

    UPDATE pool_squares
    SET state           = _state,
        claimant        = _claimant,
        user_id         = _user_id,
        modified        = (now() at time zone 'utc')
    WHERE id = _id;

    INSERT INTO pool_squares_logs (pool_square_id, user_id, state, claimant, note, remote_addr)
    VALUES (_id, _user_id, _state, _claimant, _note, _remote_addr);

    RETURN TRUE;
END;
$$;

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
