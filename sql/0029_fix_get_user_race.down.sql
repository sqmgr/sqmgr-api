-- Revert get_user to original SELECT-then-INSERT implementation
CREATE OR REPLACE FUNCTION get_user(_store stores, _store_id text) RETURNS users
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
