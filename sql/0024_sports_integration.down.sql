-- Rollback sports integration

-- Remove grid columns
ALTER TABLE grids DROP COLUMN IF EXISTS payout_config;
DROP INDEX IF EXISTS idx_grids_sports_event_id;
ALTER TABLE grids DROP COLUMN IF EXISTS sports_event_id;

-- Drop sync log table
DROP TABLE IF EXISTS sports_sync_log;

-- Drop events table and sequence
DROP INDEX IF EXISTS idx_sports_events_date;
DROP INDEX IF EXISTS idx_sports_events_status;
DROP INDEX IF EXISTS idx_sports_events_league_date;
DROP INDEX IF EXISTS idx_sports_events_espn_id;
DROP TABLE IF EXISTS sports_events;
DROP SEQUENCE IF EXISTS sports_events_id_seq;

-- Drop teams table
DROP INDEX IF EXISTS idx_sports_teams_league;
DROP TABLE IF EXISTS sports_teams;

-- Drop league enum
DROP TYPE IF EXISTS sports_league;
