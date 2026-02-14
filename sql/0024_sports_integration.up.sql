-- Sports API integration tables (ESPN)

-- League enum
CREATE TYPE sports_league AS ENUM ('nfl', 'nba', 'wnba', 'ncaab', 'ncaaf');

-- Teams table with composite PK (ESPN uses string IDs)
CREATE TABLE sports_teams (
    id VARCHAR(50) NOT NULL,
    league sports_league NOT NULL,
    name VARCHAR(100) NOT NULL,
    full_name VARCHAR(150) NOT NULL,
    abbreviation VARCHAR(10) NOT NULL,
    conference VARCHAR(50),
    division VARCHAR(50),
    location VARCHAR(100),
    created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    modified TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    PRIMARY KEY (id, league)
);

CREATE INDEX idx_sports_teams_league ON sports_teams(league);

-- Events/games table with auto-generated ID and ESPN ID
CREATE SEQUENCE sports_events_id_seq;

CREATE TABLE sports_events (
    id BIGINT PRIMARY KEY DEFAULT nextval('sports_events_id_seq'),
    espn_id VARCHAR(50),
    league sports_league NOT NULL,
    home_team_id VARCHAR(50) NOT NULL,
    away_team_id VARCHAR(50) NOT NULL,
    event_date TIMESTAMP NOT NULL,
    season INT NOT NULL,
    week INT,
    postseason BOOLEAN NOT NULL DEFAULT FALSE,
    venue VARCHAR(200),
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    period INT,
    home_score INT,
    away_score INT,
    home_q1 INT,
    home_q2 INT,
    home_q3 INT,
    home_q4 INT,
    home_ot INT,
    away_q1 INT,
    away_q2 INT,
    away_q3 INT,
    away_q4 INT,
    away_ot INT,
    created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    modified TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    last_synced TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    CONSTRAINT sports_events_home_team_id_fkey
        FOREIGN KEY (home_team_id, league) REFERENCES sports_teams(id, league)
        ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT sports_events_away_team_id_fkey
        FOREIGN KEY (away_team_id, league) REFERENCES sports_teams(id, league)
        ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
);

ALTER SEQUENCE sports_events_id_seq OWNED BY sports_events.id;

CREATE UNIQUE INDEX idx_sports_events_espn_id ON sports_events(espn_id) WHERE espn_id IS NOT NULL;
CREATE INDEX idx_sports_events_league_date ON sports_events(league, event_date);
CREATE INDEX idx_sports_events_status ON sports_events(status);
CREATE INDEX idx_sports_events_date ON sports_events(event_date);

-- Grid columns
ALTER TABLE grids ADD COLUMN sports_event_id BIGINT REFERENCES sports_events(id);
CREATE INDEX idx_grids_sports_event_id ON grids(sports_event_id);
ALTER TABLE grids ADD COLUMN payout_config TEXT;

-- Sync tracking table
CREATE TABLE sports_sync_log (
    id SERIAL PRIMARY KEY,
    sync_type VARCHAR(50) NOT NULL,
    league sports_league,
    started_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    completed_at TIMESTAMP,
    records_processed INT DEFAULT 0,
    error_message TEXT,
    success BOOLEAN
);
