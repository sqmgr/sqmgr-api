-- Copyright (C) 2026 Tom Peters
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

-- Records every write action a site admin performs through the admin API.
CREATE TABLE admin_audit_log (
    id            BIGSERIAL PRIMARY KEY,
    admin_user_id BIGINT    NOT NULL REFERENCES users (id),
    action        TEXT      NOT NULL,
    target_type   TEXT      NOT NULL,
    target_id     TEXT      NOT NULL,
    target_label  TEXT,
    details       JSONB,
    reason        TEXT,
    created       TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);

CREATE INDEX admin_audit_log_target_idx ON admin_audit_log (target_type, target_id, id DESC);
CREATE INDEX admin_audit_log_action_idx ON admin_audit_log (action, id DESC);

-- When set, the sports sync job leaves the event's status and scores alone so
-- that a manual correction made by a site admin is not overwritten.
ALTER TABLE sports_events ADD COLUMN manual_override BOOLEAN NOT NULL DEFAULT false;

-- The admin sports status page asks for the latest run per (type, league).
CREATE INDEX sports_sync_log_latest_idx ON sports_sync_log (sync_type, league, id DESC);

COMMIT;
