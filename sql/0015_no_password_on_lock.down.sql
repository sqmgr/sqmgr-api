BEGIN;

ALTER TABLE pools
DROP COLUMN open_access_on_lock;

COMMIT;