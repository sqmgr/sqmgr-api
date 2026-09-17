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
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/sportsync"
)

const defaultSyncRunsLimit = 50
const maxSyncRunsLimit = 200

// manualSyncTimeout bounds how long a manual sync started from the admin API
// may run. Schedule syncs for college basketball walk every team, so this is
// generous.
const manualSyncTimeout = 45 * time.Minute

// validSyncTypes is the set of sync types an admin may start
var validSyncTypes = map[model.SportsSyncType]bool{
	model.SportsSyncTypeTeams:    true,
	model.SportsSyncTypeSchedule: true,
	model.SportsSyncTypeScores:   true,
}

// getAdminSportsStatusEndpoint summarizes the health of the sports data
func (s *Server) getAdminSportsStatusEndpoint() http.HandlerFunc {
	type response struct {
		LastRuns    []*model.SportsSyncRun    `json:"lastRuns"`
		Running     *SyncRunStatus            `json:"running"`
		StaleEvents []*model.StaleLinkedEvent `json:"staleEvents"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		lastRuns, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) ([]*model.SportsSyncRun, error) {
			return a.LatestSportsSyncRuns(r.Context())
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		stale, err := s.model.StaleLinkedEvents(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			LastRuns:    lastRuns,
			Running:     s.syncRuns.status(),
			StaleEvents: stale,
		})
	}
}

// getAdminSportsSyncRunsEndpoint lists recent sync runs
func (s *Server) getAdminSportsSyncRunsEndpoint() http.HandlerFunc {
	type response struct {
		Runs []*model.SportsSyncRun `json:"runs"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		syncType := r.FormValue("syncType")
		if syncType != "" && !validSyncTypes[model.SportsSyncType(syncType)] {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid syncType %q", syncType))
			return
		}

		_, limit := parsePagination(r, defaultSyncRunsLimit, maxSyncRunsLimit)

		runs, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) ([]*model.SportsSyncRun, error) {
			return a.SportsSyncRunsByType(r.Context(), syncType, limit)
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{Runs: runs})
	}
}

// postAdminSportsSyncEndpoint starts a sports sync in the background
func (s *Server) postAdminSportsSyncEndpoint() http.HandlerFunc {
	type payload struct {
		SyncType string `json:"syncType"`
		League   string `json:"league"`
	}
	type response struct {
		Status string `json:"status"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		var req payload
		if ok := s.parseJSONPayload(w, r, &req); !ok {
			return
		}

		syncType := model.SportsSyncType(req.SyncType)
		if !validSyncTypes[syncType] {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid syncType %q", req.SyncType))
			return
		}

		leagues := sportsync.AllLeagues()
		var league *model.SportsLeague
		if req.League != "" {
			if !model.IsValidSportsLeague(req.League) {
				s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid league %q", req.League))
				return
			}
			l := model.SportsLeague(req.League)
			league = &l
			leagues = []model.SportsLeague{l}
		}

		if s.syncer == nil {
			s.writeErrorResponse(w, http.StatusServiceUnavailable, errors.New("sports sync is not configured"))
			return
		}

		status := SyncRunStatus{SyncType: syncType, League: league, StartedAt: time.Now()}
		if !s.syncRuns.start(status) {
			s.writeErrorResponse(w, http.StatusConflict, errors.New("a sports sync is already running"))
			return
		}

		details := map[string]interface{}{"syncType": string(syncType)}
		if league != nil {
			details["league"] = string(*league)
		}
		s.recordAudit(r.Context(), admin, model.AdminAuditRecord{
			Action:      model.AdminAuditSportsSync,
			TargetType:  model.AdminAuditTargetSportsSync,
			TargetID:    string(syncType),
			TargetLabel: fmt.Sprintf("%s sync", syncType),
			Details:     details,
		})

		go s.runManualSync(syncType, leagues)

		s.writeJSONResponse(w, http.StatusAccepted, response{Status: "started"})
	}
}

// runManualSync executes a sync outside the request that started it.
func (s *Server) runManualSync(syncType model.SportsSyncType, leagues []model.SportsLeague) {
	defer s.syncRuns.finish()

	entry := logrus.WithFields(logrus.Fields{"syncType": syncType, "leagues": leagues, "manual": true})

	// This goroutine is outside net/http's recovery, so a panic in the
	// syncer must not take the whole API down.
	defer func() {
		if p := recover(); p != nil {
			entry.WithField("panic", p).Error("manual sports sync panicked")
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), manualSyncTimeout)
	defer cancel()

	entry.Info("starting manual sports sync")

	var err error
	switch syncType {
	case model.SportsSyncTypeTeams:
		err = s.syncer.SyncTeams(ctx, leagues)
	case model.SportsSyncTypeSchedule:
		err = s.syncer.SyncSchedule(ctx, leagues)
	case model.SportsSyncTypeScores:
		err = s.syncer.SyncScores(ctx, leagues)
	}

	if err != nil {
		entry.WithError(err).Error("manual sports sync failed")
		return
	}
	entry.Info("manual sports sync finished")
}

// adminEventFromRequest loads the event named in the route, writing a 400 or
// 404 when it cannot.
func (s *Server) adminEventFromRequest(w http.ResponseWriter, r *http.Request) (*model.SportsEvent, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid event ID"))
		return nil, false
	}

	event, err := s.model.SportsEventByIDWithTeams(r.Context(), id)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if event == nil {
		s.writeErrorResponse(w, http.StatusNotFound, nil)
		return nil, false
	}

	return event, true
}

// eventLabel describes an event for the audit log.
func eventLabel(event *model.SportsEvent) string {
	if event.Name != nil && *event.Name != "" {
		return *event.Name
	}
	away, home := event.AwayTeamID, event.HomeTeamID
	if t := event.AwayTeam(); t != nil {
		away = t.FullName
	}
	if t := event.HomeTeam(); t != nil {
		home = t.FullName
	}
	return fmt.Sprintf("%s at %s", away, home)
}

// postAdminEventRefreshEndpoint fetches one event from ESPN right now
func (s *Server) postAdminEventRefreshEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		event, ok := s.adminEventFromRequest(w, r)
		if !ok {
			return
		}

		if event.ManualOverride {
			s.writeErrorResponse(w, http.StatusConflict, errors.New("event has a manual override; clear it before refreshing"))
			return
		}

		if s.syncer == nil {
			s.writeErrorResponse(w, http.StatusServiceUnavailable, errors.New("sports sync is not configured"))
			return
		}

		updated, err := s.syncer.RefreshEvent(r.Context(), event)
		if err != nil {
			if errors.Is(err, sportsync.ErrManualOverride) {
				s.writeErrorResponse(w, http.StatusConflict, err)
				return
			}
			logrus.WithError(err).WithField("eventID", event.ID).Error("could not refresh event from ESPN")
			s.writeErrorResponse(w, http.StatusBadGateway, errors.New("could not fetch the event from ESPN"))
			return
		}

		s.recordAudit(r.Context(), admin, model.AdminAuditRecord{
			Action:      model.AdminAuditEventRefresh,
			TargetType:  model.AdminAuditTargetEvent,
			TargetID:    strconv.FormatInt(event.ID, 10),
			TargetLabel: eventLabel(updated),
			Details: map[string]interface{}{
				"status":    updated.Status,
				"homeScore": updated.HomeScore,
				"awayScore": updated.AwayScore,
			},
		})

		s.writeJSONResponse(w, http.StatusOK, updated.JSON())
	}
}

// eventOverridePayload is the body of a manual score correction
type eventOverridePayload struct {
	Status       string `json:"status"`
	HomeScore    *int   `json:"homeScore"`
	AwayScore    *int   `json:"awayScore"`
	HomeQuarters []*int `json:"homeQuarters"`
	AwayQuarters []*int `json:"awayQuarters"`
	HomeOT       *int   `json:"homeOT"`
	AwayOT       *int   `json:"awayOT"`
	Reason       string `json:"reason"`
}

// toOverride validates the payload and converts it to a model override.
func (p eventOverridePayload) toOverride() (model.SportsEventOverride, error) {
	o := model.SportsEventOverride{Status: model.SportsEventStatus(p.Status)}
	if !o.Status.IsValid() {
		return o, fmt.Errorf("invalid status %q", p.Status)
	}

	for _, score := range []*int{p.HomeScore, p.AwayScore, p.HomeOT, p.AwayOT} {
		if score != nil && *score < 0 {
			return o, errors.New("scores cannot be negative")
		}
	}
	for _, quarters := range [][]*int{p.HomeQuarters, p.AwayQuarters} {
		if len(quarters) != 0 && len(quarters) != 4 {
			return o, errors.New("quarter scores must have exactly 4 entries")
		}
		for _, q := range quarters {
			if q != nil && *q < 0 {
				return o, errors.New("scores cannot be negative")
			}
		}
	}

	o.HomeScore, o.AwayScore = p.HomeScore, p.AwayScore
	o.HomeOT, o.AwayOT = p.HomeOT, p.AwayOT
	if len(p.HomeQuarters) == 4 {
		o.HomeQ1, o.HomeQ2, o.HomeQ3, o.HomeQ4 = p.HomeQuarters[0], p.HomeQuarters[1], p.HomeQuarters[2], p.HomeQuarters[3]
	}
	if len(p.AwayQuarters) == 4 {
		o.AwayQ1, o.AwayQ2, o.AwayQ3, o.AwayQ4 = p.AwayQuarters[0], p.AwayQuarters[1], p.AwayQuarters[2], p.AwayQuarters[3]
	}

	return o, nil
}

// postAdminEventOverrideEndpoint applies a manual status and score correction
func (s *Server) postAdminEventOverrideEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		var req eventOverridePayload
		if ok := s.parseJSONPayload(w, r, &req); !ok {
			return
		}

		override, err := req.toOverride()
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		event, ok := s.adminEventFromRequest(w, r)
		if !ok {
			return
		}

		previous := map[string]interface{}{
			"status":    event.Status,
			"homeScore": event.HomeScore,
			"awayScore": event.AwayScore,
		}

		if err := event.ApplyOverride(r.Context(), override); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.recordAudit(r.Context(), admin, model.AdminAuditRecord{
			Action:      model.AdminAuditEventOverride,
			TargetType:  model.AdminAuditTargetEvent,
			TargetID:    strconv.FormatInt(event.ID, 10),
			TargetLabel: eventLabel(event),
			Reason:      req.Reason,
			Details: map[string]interface{}{
				"previous": previous,
				"new": map[string]interface{}{
					"status":    event.Status,
					"homeScore": event.HomeScore,
					"awayScore": event.AwayScore,
				},
			},
		})

		s.writeJSONResponse(w, http.StatusOK, event.JSON())
	}
}

// deleteAdminEventOverrideEndpoint clears a manual override so the sync job
// resumes updating the event
func (s *Server) deleteAdminEventOverrideEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		event, ok := s.adminEventFromRequest(w, r)
		if !ok {
			return
		}

		if err := event.ClearOverride(r.Context()); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.recordAudit(r.Context(), admin, model.AdminAuditRecord{
			Action:      model.AdminAuditEventClearOverride,
			TargetType:  model.AdminAuditTargetEvent,
			TargetID:    strconv.FormatInt(event.ID, 10),
			TargetLabel: eventLabel(event),
		})

		w.WriteHeader(http.StatusNoContent)
	}
}
