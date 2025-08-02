BEGIN;

ALTER TABLE pools
    ADD COLUMN password_required bool NOT NULL DEFAULT true;

COMMIT;