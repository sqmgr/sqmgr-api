BEGIN;

ALTER TABLE pools
    DROP COLUMN password_required;

COMMIT;