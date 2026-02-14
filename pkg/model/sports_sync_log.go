/*
Copyright (C) 2019 Tom Peters

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package model

import (
	"context"
	"time"
)

// SportsSyncType represents the type of sports sync operation
type SportsSyncType string

const (
	// SportsSyncTypeTeams syncs team data
	SportsSyncTypeTeams SportsSyncType = "teams"
	// SportsSyncTypeSchedule syncs game schedule
	SportsSyncTypeSchedule SportsSyncType = "schedule"
	// SportsSyncTypeScores syncs live/recent scores
	SportsSyncTypeScores SportsSyncType = "scores"
)

// SportsSyncLog represents a sync operation log entry
type SportsSyncLog struct {
	model *Model

	ID               int64
	SyncType         SportsSyncType
	League           *SportsLeague
	StartedAt        time.Time
	CompletedAt      *time.Time
	RecordsProcessed int
	ErrorMessage     *string
	Success          *bool
}

// StartSportsSync creates a new sync log entry
func (m *Model) StartSportsSync(ctx context.Context, syncType SportsSyncType, league *SportsLeague) (*SportsSyncLog, error) {
	log := &SportsSyncLog{
		model:    m,
		SyncType: syncType,
		League:   league,
	}

	const query = `
		INSERT INTO sports_sync_log (sync_type, league, started_at)
		VALUES ($1, $2, (NOW() AT TIME ZONE 'utc'))
		RETURNING id, started_at
	`

	err := m.DB.QueryRowContext(ctx, query, syncType, league).Scan(&log.ID, &log.StartedAt)
	if err != nil {
		return nil, err
	}

	return log, nil
}

// Complete marks the sync as completed
func (l *SportsSyncLog) Complete(ctx context.Context, recordsProcessed int, success bool, errorMessage string) error {
	l.RecordsProcessed = recordsProcessed
	l.Success = &success

	var errMsg *string
	if errorMessage != "" {
		errMsg = &errorMessage
	}
	l.ErrorMessage = errMsg

	const query = `
		UPDATE sports_sync_log
		SET completed_at = (NOW() AT TIME ZONE 'utc'),
		    records_processed = $1,
		    success = $2,
		    error_message = $3
		WHERE id = $4
	`

	_, err := l.model.DB.ExecContext(ctx, query, recordsProcessed, success, errMsg, l.ID)
	return err
}

// LastSuccessfulSportsSync returns the last successful sync for a given type and optional league
func (m *Model) LastSuccessfulSportsSync(ctx context.Context, syncType SportsSyncType, league *SportsLeague) (*SportsSyncLog, error) {
	log := &SportsSyncLog{model: m}

	var query string
	var args []interface{}

	if league == nil {
		query = `
			SELECT id, sync_type, league, started_at, completed_at, records_processed, error_message, success
			FROM sports_sync_log
			WHERE sync_type = $1 AND success = true
			ORDER BY completed_at DESC
			LIMIT 1
		`
		args = []interface{}{syncType}
	} else {
		query = `
			SELECT id, sync_type, league, started_at, completed_at, records_processed, error_message, success
			FROM sports_sync_log
			WHERE sync_type = $1 AND league = $2 AND success = true
			ORDER BY completed_at DESC
			LIMIT 1
		`
		args = []interface{}{syncType, *league}
	}

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&log.ID,
		&log.SyncType,
		&log.League,
		&log.StartedAt,
		&log.CompletedAt,
		&log.RecordsProcessed,
		&log.ErrorMessage,
		&log.Success,
	)
	if err != nil {
		return nil, err
	}

	return log, nil
}

// Deprecated aliases for backward compatibility
type BDLSyncType = SportsSyncType
type BDLSyncLog = SportsSyncLog

const (
	BDLSyncTypeTeams    = SportsSyncTypeTeams
	BDLSyncTypeSchedule = SportsSyncTypeSchedule
	BDLSyncTypeScores   = SportsSyncTypeScores
)

func (m *Model) StartBDLSync(ctx context.Context, syncType BDLSyncType, league *BDLLeague) (*BDLSyncLog, error) {
	return m.StartSportsSync(ctx, syncType, league)
}

func (m *Model) LastSuccessfulSync(ctx context.Context, syncType BDLSyncType, league *BDLLeague) (*BDLSyncLog, error) {
	return m.LastSuccessfulSportsSync(ctx, syncType, league)
}
