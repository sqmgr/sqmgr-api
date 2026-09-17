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

// Package sportsync pulls teams, schedules, and scores from ESPN into the
// database. It is used by the sqmgr-sports-sync command and by the admin API
// for on-demand syncs.
package sportsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/sports"
)

// ErrManualOverride is returned when an event cannot be refreshed because a
// site admin has taken manual control of it.
var ErrManualOverride = errors.New("sportsync: event has a manual override")

// syncLogWriteTimeout bounds the detached write that closes a sync log row.
const syncLogWriteTimeout = 10 * time.Second

// Client is the subset of the ESPN client the syncer depends on.
type Client interface {
	GetTeams(ctx context.Context, league sports.League) ([]sports.Team, error)
	GetSeasonInfo(ctx context.Context, league sports.League) (*sports.SeasonInfo, error)
	GetTeamSchedule(ctx context.Context, league sports.League, teamID string, seasonType sports.SeasonType) ([]sports.Event, error)
	GetScoreboard(ctx context.Context, league sports.League, opts sports.ScoreboardOptions) ([]sports.Event, error)
	GetScoreboardForDateRange(ctx context.Context, league sports.League, startDate, endDate time.Time) ([]sports.Event, error)
	GetEventSummary(ctx context.Context, league sports.League, eventID string) (*sports.Event, error)
}

// Syncer copies ESPN data into the database.
type Syncer struct {
	model  *model.Model
	client Client
	log    *logrus.Entry
	dryRun bool
}

// Options configures a Syncer.
type Options struct {
	// Logger receives progress messages. Defaults to the standard logger.
	Logger *logrus.Entry
	// DryRun logs what would change without writing to the database.
	DryRun bool
}

// New returns a Syncer backed by the given model and ESPN client.
func New(m *model.Model, client Client, opts Options) *Syncer {
	log := opts.Logger
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}
	if opts.DryRun {
		log = log.WithField("dry-run", true)
	}
	return &Syncer{model: m, client: client, log: log, dryRun: opts.DryRun}
}

// AllLeagues is every league the syncer knows how to handle.
func AllLeagues() []model.SportsLeague {
	return []model.SportsLeague{
		model.SportsLeagueNFL,
		model.SportsLeagueNBA,
		model.SportsLeagueWNBA,
		model.SportsLeagueNCAAB,
		model.SportsLeagueNCAAF,
	}
}

// SyncTeams fetches and upserts every team for the given leagues.
func (s *Syncer) SyncTeams(ctx context.Context, leagues []model.SportsLeague) error {
	s.log.Info("syncing teams")

	for _, league := range leagues {
		leagueLog := s.log.WithField("league", league)
		leagueLog.Info("fetching teams")

		syncLog := s.startSyncLog(ctx, model.SportsSyncTypeTeams, &league, leagueLog)

		teams, err := s.client.GetTeams(ctx, sports.League(league))
		if err != nil {
			s.completeSyncLog(ctx, syncLog, 0, err)
			return fmt.Errorf("fetching teams for %s: %w", league, err)
		}

		leagueLog.WithField("count", len(teams)).Info("found teams")

		for _, team := range teams {
			teamLog := leagueLog.WithFields(logrus.Fields{
				"teamID": team.ID,
				"name":   team.Name,
			})

			teamLog.Debug("processing team")

			if s.dryRun {
				continue
			}

			if err := s.model.UpsertSportsTeam(ctx, nil, s.teamFromESPN(league, team)); err != nil {
				teamLog.WithError(err).Error("failed to upsert team")
				continue
			}
		}

		s.completeSyncLog(ctx, syncLog, len(teams), nil)
	}

	return nil
}

// SyncSchedule fetches the season schedule for the given leagues and upserts
// every event.
func (s *Syncer) SyncSchedule(ctx context.Context, leagues []model.SportsLeague) error {
	s.log.Info("syncing schedule")

	now := time.Now()

	for _, league := range leagues {
		leagueLog := s.log.WithField("league", league)

		// Get season info to determine date range
		seasonInfo, err := s.client.GetSeasonInfo(ctx, sports.League(league))
		if err != nil {
			leagueLog.WithError(err).Error("failed to get season info")
			continue
		}

		leagueLog.WithFields(logrus.Fields{
			"seasonYear":  seasonInfo.Year,
			"seasonStart": seasonInfo.StartDate.Format("2006-01-02"),
			"seasonEnd":   seasonInfo.EndDate.Format("2006-01-02"),
			"inSeason":    seasonInfo.InSeason,
			"seasonType":  seasonInfo.Type,
		}).Info("got season info")

		syncLog := s.startSyncLog(ctx, model.SportsSyncTypeSchedule, &league, leagueLog)

		var events []sports.Event

		// Use different fetching strategies based on league
		switch league {
		case model.SportsLeagueNFL, model.SportsLeagueNCAAF:
			// For football, use week-based fetching
			events, err = s.footballSchedule(ctx, league, seasonInfo.Year, leagueLog)
			if err != nil {
				s.completeSyncLog(ctx, syncLog, 0, err)
				leagueLog.WithError(err).Error("failed to sync football schedule")
				continue
			}

		case model.SportsLeagueNCAAB:
			// For college basketball, use team-based fetching to get all games.
			// ESPN's scoreboard only returns curated games, missing smaller schools.
			events, err = s.teamSchedules(ctx, league, leagueLog)
			if err != nil {
				s.completeSyncLog(ctx, syncLog, 0, err)
				leagueLog.WithError(err).Error("failed to sync team schedules")
				continue
			}

		default:
			// For NBA/WNBA, use date range (all games appear in scoreboard)
			var startDate, endDate time.Time
			if seasonInfo.InSeason {
				startDate = now
				endDate = seasonInfo.EndDate
			} else {
				startDate = seasonInfo.StartDate
				endDate = seasonInfo.EndDate
			}

			leagueLog.WithFields(logrus.Fields{
				"startDate": startDate.Format("2006-01-02"),
				"endDate":   endDate.Format("2006-01-02"),
			}).Info("fetching schedule by date range")

			events, err = s.client.GetScoreboardForDateRange(ctx, sports.League(league), startDate, endDate)
			if err != nil {
				s.completeSyncLog(ctx, syncLog, 0, err)
				leagueLog.WithError(err).Error("failed to fetch schedule")
				continue
			}
		}

		leagueLog.WithField("count", len(events)).Info("found events")

		processedCount := 0
		for _, event := range events {
			if err := s.ProcessEvent(ctx, league, event); err != nil {
				leagueLog.WithError(err).WithField("eventID", event.ID).Error("failed to process event")
				continue
			}
			processedCount++
		}

		leagueLog.WithField("processedCount", processedCount).Info("finished syncing schedule")

		s.completeSyncLog(ctx, syncLog, processedCount, nil)
	}

	return nil
}

// teamSchedules fetches the schedule of every team in the league and
// deduplicates the events.
func (s *Syncer) teamSchedules(ctx context.Context, league model.SportsLeague, leagueLog *logrus.Entry) ([]sports.Event, error) {
	// Get all teams for this league from the database
	teams, err := s.model.SportsTeamsByLeague(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("fetching teams from database: %w", err)
	}

	if len(teams) == 0 {
		return nil, fmt.Errorf("no teams found in database for %s - sync teams first", league)
	}

	leagueLog.WithField("teamCount", len(teams)).Info("fetching schedules for all teams")

	var allEvents []sports.Event
	seenIDs := make(map[string]bool)

	// Fetch both regular season and postseason schedules for each team
	seasonTypes := []sports.SeasonType{sports.SeasonTypeRegular, sports.SeasonTypePostseason}

	for _, seasonType := range seasonTypes {
		leagueLog.WithField("seasonType", seasonType).Info("fetching team schedules")

		for i, team := range teams {
			teamLog := leagueLog.WithFields(logrus.Fields{
				"teamID":     team.ID,
				"teamName":   team.Name,
				"seasonType": seasonType,
				"progress":   fmt.Sprintf("%d/%d", i+1, len(teams)),
			})

			teamLog.Debug("fetching team schedule")

			events, err := s.client.GetTeamSchedule(ctx, sports.League(league), team.ID, seasonType)
			if err != nil {
				teamLog.WithError(err).Warn("failed to fetch team schedule")
				continue
			}

			// Deduplicate events (each game appears in both teams' schedules)
			for _, e := range events {
				if !seenIDs[e.ID] {
					seenIDs[e.ID] = true
					allEvents = append(allEvents, e)
				}
			}

			// Log progress every 50 teams
			if (i+1)%50 == 0 {
				leagueLog.WithFields(logrus.Fields{
					"progress":    fmt.Sprintf("%d/%d", i+1, len(teams)),
					"eventsFound": len(allEvents),
				}).Info("sync progress")
			}
		}
	}

	return allEvents, nil
}

// footballSchedule fetches NFL or NCAAF schedules week by week.
func (s *Syncer) footballSchedule(ctx context.Context, league model.SportsLeague, season int, leagueLog *logrus.Entry) ([]sports.Event, error) {
	var events []sports.Event
	var sportsLeague sports.League

	switch league {
	case model.SportsLeagueNFL:
		sportsLeague = sports.LeagueNFL
	case model.SportsLeagueNCAAF:
		sportsLeague = sports.LeagueNCAAF
	default:
		return nil, fmt.Errorf("unsupported football league: %s", league)
	}

	// Fetch regular season weeks 1-18 (NFL) or 1-15 (NCAAF)
	maxWeek := 18
	if league == model.SportsLeagueNCAAF {
		maxWeek = 15
	}

	for week := 1; week <= maxWeek; week++ {
		weekEvents, err := s.client.GetScoreboard(ctx, sportsLeague, sports.ScoreboardOptions{
			Season:     season,
			Week:       week,
			SeasonType: sports.SeasonTypeRegular,
		})
		if err != nil {
			leagueLog.WithError(err).WithFields(logrus.Fields{"week": week, "seasonType": "regular"}).Warn("failed to fetch week")
			continue
		}
		events = append(events, weekEvents...)
	}

	// Fetch postseason weeks
	postseasonWeeks := 5 // NFL: Wild Card, Divisional, Conference, Pro Bowl, Super Bowl
	if league == model.SportsLeagueNCAAF {
		postseasonWeeks = 3 // NCAAF has fewer postseason weeks
	}

	for week := 1; week <= postseasonWeeks; week++ {
		weekEvents, err := s.client.GetScoreboard(ctx, sportsLeague, sports.ScoreboardOptions{
			Season:     season,
			Week:       week,
			SeasonType: sports.SeasonTypePostseason,
		})
		if err != nil {
			leagueLog.WithError(err).WithFields(logrus.Fields{"week": week, "seasonType": "postseason"}).Warn("failed to fetch week")
			continue
		}
		events = append(events, weekEvents...)
	}

	return events, nil
}

// SyncScores refreshes every event that is in progress or about to start for
// the given leagues.
func (s *Syncer) SyncScores(ctx context.Context, leagues []model.SportsLeague) error {
	s.log.Info("syncing scores")

	syncLog := s.startSyncLog(ctx, model.SportsSyncTypeScores, nil, s.log)

	// Finalize any stale events that fell out of the score update window
	if !s.dryRun {
		staleCount, err := s.model.FinalizeStaleEvents(ctx)
		if err != nil {
			s.log.WithError(err).Warn("failed to finalize stale events")
		} else if staleCount > 0 {
			s.log.WithField("count", staleCount).Info("finalized stale events")
		}
	}

	// Find events that need score updates
	events, err := s.model.EventsNeedingScoreUpdate(ctx)
	if err != nil {
		s.completeSyncLog(ctx, syncLog, 0, err)
		return fmt.Errorf("querying events needing update: %w", err)
	}

	// Build a set of leagues to filter by
	leagueSet := make(map[model.SportsLeague]bool)
	for _, l := range leagues {
		leagueSet[l] = true
	}

	// Group events by league for efficient API calls, filtering by specified leagues
	eventsByLeague := make(map[model.SportsLeague][]*model.SportsEvent)
	for _, event := range events {
		if leagueSet[event.League] {
			eventsByLeague[event.League] = append(eventsByLeague[event.League], event)
		}
	}

	// Count total events after filtering
	totalEvents := 0
	for _, leagueEvents := range eventsByLeague {
		totalEvents += len(leagueEvents)
	}

	s.log.WithField("count", totalEvents).Info("found events needing score update")

	updatedCount := 0
	for league, leagueEvents := range eventsByLeague {
		leagueLog := s.log.WithField("league", league)

		// Fetch today's and yesterday's scoreboards for this league
		now := time.Now()
		dates := []string{
			now.Format("20060102"),
			now.AddDate(0, 0, -1).Format("20060102"),
		}

		// Build lookup map by ESPN ID from both days
		espnEventMap := make(map[string]sports.Event)
		for _, date := range dates {
			scoreboardEvents, err := s.client.GetScoreboard(ctx, sports.League(league), sports.ScoreboardOptions{
				Date: date,
			})
			if err != nil {
				leagueLog.WithError(err).WithField("date", date).Warn("failed to fetch scoreboard")
				continue
			}
			for _, e := range scoreboardEvents {
				espnEventMap[e.ID] = e
			}
		}

		// Update each event that needs it
		for _, dbEvent := range leagueEvents {
			eventLog := leagueLog.WithFields(logrus.Fields{
				"eventID": dbEvent.ID,
				"espnID":  dbEvent.ESPNID,
				"status":  dbEvent.Status,
			})

			espnEvent, found := espnEventMap[dbEvent.ESPNID]
			if !found {
				// Event not in scoreboard - fetch it individually via summary endpoint.
				// This handles smaller school games that ESPN doesn't feature in daily scoreboards.
				eventLog.Debug("event not found in scoreboard, fetching via summary endpoint")
				fetchedEvent, err := s.client.GetEventSummary(ctx, sports.League(league), dbEvent.ESPNID)
				if err != nil {
					eventLog.WithError(err).Debug("failed to fetch event summary")
					continue
				}
				espnEvent = *fetchedEvent
			}

			if err := s.ProcessEvent(ctx, league, espnEvent); err != nil {
				eventLog.WithError(err).Error("failed to update event")
				continue
			}

			updatedCount++
		}
	}

	s.log.WithField("updatedCount", updatedCount).Info("finished syncing scores")

	s.completeSyncLog(ctx, syncLog, updatedCount, nil)

	return nil
}

// RefreshEvent fetches one event from ESPN and updates it. It returns the
// reloaded event. Events under manual override are left alone and
// ErrManualOverride is returned.
func (s *Syncer) RefreshEvent(ctx context.Context, event *model.SportsEvent) (*model.SportsEvent, error) {
	if event.ManualOverride {
		return nil, ErrManualOverride
	}
	if event.ESPNID == "" {
		return nil, fmt.Errorf("refreshing event %d: no ESPN id", event.ID)
	}

	fetched, err := s.client.GetEventSummary(ctx, sports.League(event.League), event.ESPNID)
	if err != nil {
		return nil, fmt.Errorf("fetching event %s from ESPN: %w", event.ESPNID, err)
	}

	if err := s.ProcessEvent(ctx, event.League, *fetched); err != nil {
		return nil, err
	}

	updated, err := s.model.SportsEventByIDWithTeams(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("reloading event %d: %w", event.ID, err)
	}

	return updated, nil
}

// ProcessEvent upserts the teams and event, notifies connected clients when
// score data changed, and syncs grid team details when the teams changed.
// Events that a site admin has manually overridden are skipped.
func (s *Syncer) ProcessEvent(ctx context.Context, league model.SportsLeague, event sports.Event) error {
	if s.dryRun {
		s.log.WithFields(logrus.Fields{
			"eventID":  event.ID,
			"homeTeam": event.HomeTeam.Abbreviation,
			"awayTeam": event.AwayTeam.Abbreviation,
			"status":   event.Status,
		}).Info("would process event")
		return nil
	}

	// Load existing event to detect team ID changes and manual overrides
	existingEvent, err := s.model.SportsEventByESPNID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("loading existing event: %w", err)
	}

	if existingEvent != nil && existingEvent.ManualOverride {
		s.log.WithFields(logrus.Fields{
			"eventID": existingEvent.ID,
			"espnID":  event.ID,
		}).Debug("skipping event with manual override")
		return nil
	}

	// Ensure teams exist
	if err := s.model.UpsertSportsTeam(ctx, nil, s.teamFromESPN(league, event.HomeTeam)); err != nil {
		return fmt.Errorf("upserting home team: %w", err)
	}
	if err := s.model.UpsertSportsTeam(ctx, nil, s.teamFromESPN(league, event.AwayTeam)); err != nil {
		return fmt.Errorf("upserting away team: %w", err)
	}

	sportsEvent := s.eventFromESPN(league, event)

	if err := s.model.UpsertSportsEvent(ctx, nil, sportsEvent); err != nil {
		return fmt.Errorf("upserting event: %w", err)
	}

	// Notify connected clients if score-relevant data changed
	if existingEvent != nil && EventDataChanged(existingEvent, sportsEvent) {
		if err := s.model.NotifySportsEventUpdated(ctx, sportsEvent.ID); err != nil {
			s.log.WithError(err).WithField("eventID", sportsEvent.ID).Warn("failed to send sports_event_updated notification")
		}
	}

	// Sync grid team names and colors only when team IDs changed (or event is new)
	teamsChanged := existingEvent == nil ||
		existingEvent.HomeTeamID != event.HomeTeam.ID ||
		existingEvent.AwayTeamID != event.AwayTeam.ID
	if teamsChanged {
		gridCount, err := s.model.SyncGridsFromEvent(ctx, sportsEvent.ID)
		if err != nil {
			return fmt.Errorf("syncing grids from event: %w", err)
		}
		if gridCount > 0 {
			s.log.WithFields(logrus.Fields{
				"eventID":      sportsEvent.ID,
				"gridsUpdated": gridCount,
			}).Info("synced grid team names/colors from event")
		}
	}

	return nil
}

// teamFromESPN builds a SportsTeam from ESPN team data.
func (s *Syncer) teamFromESPN(league model.SportsLeague, team sports.Team) *model.SportsTeam {
	sportsTeam := s.model.NewSportsTeam()
	sportsTeam.ID = team.ID
	sportsTeam.League = league
	sportsTeam.Name = team.Name
	sportsTeam.FullName = team.DisplayName
	sportsTeam.Abbreviation = team.Abbreviation
	if team.Location != "" {
		sportsTeam.Location = &team.Location
	}
	if team.Color != "" {
		sportsTeam.Color = &team.Color
	}
	if team.AlternateColor != "" {
		sportsTeam.AlternateColor = &team.AlternateColor
	}
	return sportsTeam
}

// eventFromESPN builds a SportsEvent from ESPN event data.
func (s *Syncer) eventFromESPN(league model.SportsLeague, event sports.Event) *model.SportsEvent {
	sportsEvent := s.model.NewSportsEvent()
	sportsEvent.ESPNID = event.ID
	sportsEvent.League = league
	if event.Name != "" {
		sportsEvent.Name = &event.Name
	}
	sportsEvent.HomeTeamID = event.HomeTeam.ID
	sportsEvent.AwayTeamID = event.AwayTeam.ID
	sportsEvent.EventDate = event.Date
	sportsEvent.Season = event.Season
	sportsEvent.Week = event.Week
	sportsEvent.Postseason = event.SeasonType == sports.SeasonTypePostseason
	if event.Venue != "" {
		sportsEvent.Venue = &event.Venue
	}

	// Map status
	switch event.Status {
	case sports.EventStatusFinal:
		sportsEvent.Status = model.SportsEventStatusFinal
	case sports.EventStatusInProgress:
		sportsEvent.Status = model.SportsEventStatusInProgress
	default:
		sportsEvent.Status = model.SportsEventStatusScheduled
	}

	period := event.Period
	sportsEvent.Period = &period
	if event.Clock != "" {
		sportsEvent.Clock = &event.Clock
	}
	if event.StatusDetail != "" {
		sportsEvent.StatusDetail = &event.StatusDetail
	}
	sportsEvent.HomeScore = event.HomeTeamScore
	sportsEvent.AwayScore = event.AwayTeamScore
	sportsEvent.HomeQ1 = event.HomeQ1
	sportsEvent.HomeQ2 = event.HomeQ2
	sportsEvent.HomeQ3 = event.HomeQ3
	sportsEvent.HomeQ4 = event.HomeQ4
	sportsEvent.HomeOT = event.HomeOT
	sportsEvent.AwayQ1 = event.AwayQ1
	sportsEvent.AwayQ2 = event.AwayQ2
	sportsEvent.AwayQ3 = event.AwayQ3
	sportsEvent.AwayQ4 = event.AwayQ4
	sportsEvent.AwayOT = event.AwayOT

	return sportsEvent
}

// startSyncLog opens a sync log entry unless running dry. A nil return means
// no entry was created, which is not fatal.
func (s *Syncer) startSyncLog(ctx context.Context, syncType model.SportsSyncType, league *model.SportsLeague, log *logrus.Entry) *model.SportsSyncLog {
	if s.dryRun {
		return nil
	}
	syncLog, err := s.model.StartSportsSync(ctx, syncType, league)
	if err != nil {
		log.WithError(err).Warn("failed to create sync log")
		return nil
	}
	return syncLog
}

// completeSyncLog closes a sync log entry, recording the error when present.
func (s *Syncer) completeSyncLog(ctx context.Context, syncLog *model.SportsSyncLog, processed int, err error) {
	if syncLog == nil {
		return
	}

	// The run may be ending because ctx expired; the log row still has to be
	// closed or the run shows as in progress forever.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), syncLogWriteTimeout)
	defer cancel()

	var completeErr error
	if err != nil {
		completeErr = syncLog.Complete(ctx, processed, false, err.Error())
	} else {
		completeErr = syncLog.Complete(ctx, processed, true, "")
	}
	if completeErr != nil {
		s.log.WithError(completeErr).WithField("syncLogID", syncLog.ID).Warn("failed to complete sync log")
	}
}

// EventDataChanged reports whether any score-relevant field differs between
// two versions of an event.
func EventDataChanged(existing, updated *model.SportsEvent) bool {
	if existing.Status != updated.Status {
		return true
	}
	if !intPtrEqual(existing.HomeScore, updated.HomeScore) || !intPtrEqual(existing.AwayScore, updated.AwayScore) {
		return true
	}
	if !intPtrEqual(existing.Period, updated.Period) {
		return true
	}
	if !strPtrEqual(existing.Clock, updated.Clock) {
		return true
	}
	if !strPtrEqual(existing.StatusDetail, updated.StatusDetail) {
		return true
	}
	if !intPtrEqual(existing.HomeQ1, updated.HomeQ1) || !intPtrEqual(existing.AwayQ1, updated.AwayQ1) {
		return true
	}
	if !intPtrEqual(existing.HomeQ2, updated.HomeQ2) || !intPtrEqual(existing.AwayQ2, updated.AwayQ2) {
		return true
	}
	if !intPtrEqual(existing.HomeQ3, updated.HomeQ3) || !intPtrEqual(existing.AwayQ3, updated.AwayQ3) {
		return true
	}
	if !intPtrEqual(existing.HomeQ4, updated.HomeQ4) || !intPtrEqual(existing.AwayQ4, updated.AwayQ4) {
		return true
	}
	if !intPtrEqual(existing.HomeOT, updated.HomeOT) || !intPtrEqual(existing.AwayOT, updated.AwayOT) {
		return true
	}
	return false
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func strPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
