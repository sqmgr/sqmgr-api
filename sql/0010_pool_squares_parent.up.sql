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
