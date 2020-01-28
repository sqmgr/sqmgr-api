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

ALTER TABLE pool_squares ADD COLUMN parent_id bigint REFERENCES pool_squares (id);

DROP FUNCTION update_pool_square(_id bigint, _state square_states, _claimant text, _user_id bigint,
    _remote_addr text, _note text, _is_admin boolean);
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
    _parent_id     integer;
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

    _parent_id = _row.parent_id;
    IF _state = 'unclaimed' THEN
        _claimant := NULL;
        _user_id := NULL;
        _parent_id := NULL;
    END IF;

    UPDATE pool_squares
    SET state           = _state,
        claimant        = _claimant,
        user_id         = _user_id,
        parent_id       = _parent_id,
        modified        = (now() at time zone 'utc')
    WHERE id = _id;

    INSERT INTO pool_squares_logs (pool_square_id, user_id, state, claimant, note, remote_addr)
    VALUES (_id, _user_id, _state, _claimant, _note, _remote_addr);

    RETURN TRUE;
END;
$$;


COMMIT;
