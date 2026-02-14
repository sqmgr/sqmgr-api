-- Fix race condition in get_user where concurrent calls for the same new user
-- could cause a duplicate key violation on users_store_store_id_key
CREATE OR REPLACE FUNCTION get_user(_store stores, _store_id text) RETURNS users
    LANGUAGE plpgsql
AS
$$
declare
    _record users;
begin
    INSERT INTO users (store, store_id)
    VALUES (_store, _store_id)
    ON CONFLICT (store, store_id) DO NOTHING;

    SELECT *
    INTO _record
    FROM users
    WHERE store = _store
      AND store_id = _store_id;

    return _record;
end;
$$;
