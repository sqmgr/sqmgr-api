/*
Copyright (C) 2026 Tom Peters

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

package server

import (
	"sync"
	"time"

	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// SyncRunStatus describes a manual sports sync that is in progress.
type SyncRunStatus struct {
	SyncType  model.SportsSyncType `json:"syncType"`
	League    *model.SportsLeague  `json:"league"`
	StartedAt time.Time            `json:"startedAt"`
}

// syncRunner allows one manual sports sync at a time.
type syncRunner struct {
	mu      sync.Mutex
	running *SyncRunStatus
}

// start records a run as in progress. It returns false when another run is
// already in progress.
func (r *syncRunner) start(status SyncRunStatus) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running != nil {
		return false
	}
	r.running = &status
	return true
}

// finish clears the in-progress run.
func (r *syncRunner) finish() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = nil
}

// status returns a copy of the in-progress run, or nil when idle.
func (r *syncRunner) status() *SyncRunStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running == nil {
		return nil
	}
	copied := *r.running
	return &copied
}
