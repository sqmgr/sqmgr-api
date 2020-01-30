BEGIN;

ALTER TABLE pools
ADD COLUMN open_access_on_lock bool NOT NULL DEFAULT false;

COMMIT;