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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/internal/database"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/sports"
)

var (
	syncTeams    = flag.Bool("sync-teams", false, "Sync teams from ESPN for all leagues")
	syncSchedule = flag.Bool("sync-schedule", false, "Sync upcoming game schedule")
	syncScores   = flag.Bool("sync-scores", false, "Sync scores for in-progress/recent games")
	dryRun       = flag.Bool("dry-run", false, "Don't persist changes to database")
	league       = flag.String("league", "", "Specific league to sync (nfl, nba, wnba, ncaab, ncaaf)")
	log          = logrus.NewEntry(logrus.StandardLogger())
)

func main() {
	flag.Parse()

	// Set log level from environment
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			log.WithError(err).Fatal("invalid LOG_LEVEL")
		}
		logrus.SetLevel(level)
	}

	if err := config.Load(); err != nil {
		log.WithError(err).Fatal("could not load config")
	}

	if *dryRun {
		log = log.WithField("dry-run", true)
	}

	log.Info("starting sports sync (ESPN)")
	defer func() {
		log.Info("finished sports sync")
	}()

	db, err := database.Open()
	if err != nil {
		log.WithError(err).Fatal("could not open database")
	}
	defer db.Close()

	m := model.New(db)

	client := sports.NewClient(sports.Config{
		Logger: log,
	})

	ctx := context.Background()

	if !*syncTeams && !*syncSchedule && !*syncScores {
		log.Fatal("must specify one of: --sync-teams, --sync-schedule, --sync-scores")
	}

	// Determine which leagues to sync
	leagues := getLeaguesToSync()

	if *syncTeams {
		if err := doSyncTeams(ctx, m, client, leagues); err != nil {
			log.WithError(err).Fatal("failed to sync teams")
		}
	}

	if *syncSchedule {
		if err := doSyncSchedule(ctx, m, client, leagues); err != nil {
			log.WithError(err).Fatal("failed to sync schedule")
		}
	}

	if *syncScores {
		if err := doSyncScores(ctx, m, client, leagues); err != nil {
			log.WithError(err).Fatal("failed to sync scores")
		}
	}
}

func getLeaguesToSync() []model.SportsLeague {
	if *league != "" {
		if !model.IsValidSportsLeague(*league) {
			log.WithField("league", *league).Fatal("invalid league")
		}
		return []model.SportsLeague{model.SportsLeague(*league)}
	}
	return []model.SportsLeague{
		model.SportsLeagueNFL,
		model.SportsLeagueNBA,
		model.SportsLeagueWNBA,
		model.SportsLeagueNCAAB,
		model.SportsLeagueNCAAF,
	}
}

func doSyncTeams(ctx context.Context, m *model.Model, client *sports.Client, leagues []model.SportsLeague) error {
	log.Info("syncing teams")

	for _, league := range leagues {
		leagueLog := log.WithField("league", league)
		leagueLog.Info("fetching teams")

		// Start sync log
		var syncLog *model.SportsSyncLog
		var err error
		if !*dryRun {
			leaguePtr := league
			syncLog, err = m.StartSportsSync(ctx, model.SportsSyncTypeTeams, &leaguePtr)
			if err != nil {
				leagueLog.WithError(err).Warn("failed to create sync log")
			}
		}

		teams, err := client.GetTeams(ctx, sports.League(league))
		if err != nil {
			if syncLog != nil {
				_ = syncLog.Complete(ctx, 0, false, err.Error())
			}
			return fmt.Errorf("fetching teams for %s: %w", league, err)
		}

		leagueLog.WithField("count", len(teams)).Info("found teams")

		for _, team := range teams {
			teamLog := leagueLog.WithFields(logrus.Fields{
				"teamID": team.ID,
				"name":   team.Name,
			})

			teamLog.Debug("processing team")

			if *dryRun {
				continue
			}

			sportsTeam := m.NewSportsTeam()
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

			if err := m.UpsertSportsTeam(ctx, nil, sportsTeam); err != nil {
				teamLog.WithError(err).Error("failed to upsert team")
				continue
			}
		}

		if syncLog != nil {
			_ = syncLog.Complete(ctx, len(teams), true, "")
		}
	}

	return nil
}

func doSyncSchedule(ctx context.Context, m *model.Model, client *sports.Client, leagues []model.SportsLeague) error {
	log.Info("syncing schedule")

	now := time.Now()

	for _, league := range leagues {
		leagueLog := log.WithField("league", league)

		// Get season info to determine date range
		seasonInfo, err := client.GetSeasonInfo(ctx, sports.League(league))
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

		// Start sync log
		var syncLog *model.SportsSyncLog
		if !*dryRun {
			leaguePtr := league
			syncLog, err = m.StartSportsSync(ctx, model.SportsSyncTypeSchedule, &leaguePtr)
			if err != nil {
				leagueLog.WithError(err).Warn("failed to create sync log")
			}
		}

		var events []sports.Event

		// Use different fetching strategies based on league
		switch league {
		case model.SportsLeagueNFL, model.SportsLeagueNCAAF:
			// For football, use week-based fetching
			events, err = syncFootballSchedule(ctx, client, league, seasonInfo.Year, leagueLog)
			if err != nil {
				if syncLog != nil {
					_ = syncLog.Complete(ctx, 0, false, err.Error())
				}
				leagueLog.WithError(err).Error("failed to sync football schedule")
				continue
			}

		case model.SportsLeagueNCAAB:
			// For college basketball, use team-based fetching to get all games
			// ESPN's scoreboard only returns curated games, missing smaller schools
			events, err = syncTeamSchedules(ctx, m, client, league, leagueLog)
			if err != nil {
				if syncLog != nil {
					_ = syncLog.Complete(ctx, 0, false, err.Error())
				}
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

			events, err = client.GetScoreboardForDateRange(ctx, sports.League(league), startDate, endDate)
			if err != nil {
				if syncLog != nil {
					_ = syncLog.Complete(ctx, 0, false, err.Error())
				}
				leagueLog.WithError(err).Error("failed to fetch schedule")
				continue
			}
		}

		leagueLog.WithField("count", len(events)).Info("found events")

		processedCount := 0
		for _, event := range events {
			if err := processEvent(ctx, m, league, event); err != nil {
				leagueLog.WithError(err).WithField("eventID", event.ID).Error("failed to process event")
				continue
			}
			processedCount++
		}

		leagueLog.WithField("processedCount", processedCount).Info("finished syncing schedule")

		if syncLog != nil {
			_ = syncLog.Complete(ctx, processedCount, true, "")
		}
	}

	return nil
}

// syncTeamSchedules fetches schedules for all teams in a league
// This is needed for college sports where ESPN's scoreboard only returns curated games
func syncTeamSchedules(ctx context.Context, m *model.Model, client *sports.Client, league model.SportsLeague, leagueLog *logrus.Entry) ([]sports.Event, error) {
	// Get all teams for this league from the database
	teams, err := m.SportsTeamsByLeague(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("fetching teams from database: %w", err)
	}

	if len(teams) == 0 {
		return nil, fmt.Errorf("no teams found in database for %s - run --sync-teams first", league)
	}

	leagueLog.WithField("teamCount", len(teams)).Info("fetching schedules for all teams")

	var allEvents []sports.Event
	seenIDs := make(map[string]bool)

	for i, team := range teams {
		teamLog := leagueLog.WithFields(logrus.Fields{
			"teamID":   team.ID,
			"teamName": team.Name,
			"progress": fmt.Sprintf("%d/%d", i+1, len(teams)),
		})

		teamLog.Debug("fetching team schedule")

		events, err := client.GetTeamSchedule(ctx, sports.League(league), team.ID)
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

	return allEvents, nil
}

// syncFootballSchedule syncs NFL or NCAAF schedules using week-based fetching
func syncFootballSchedule(ctx context.Context, client *sports.Client, league model.SportsLeague, season int, leagueLog *logrus.Entry) ([]sports.Event, error) {
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
		weekEvents, err := client.GetScoreboard(ctx, sportsLeague, sports.ScoreboardOptions{
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
		weekEvents, err := client.GetScoreboard(ctx, sportsLeague, sports.ScoreboardOptions{
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

func doSyncScores(ctx context.Context, m *model.Model, client *sports.Client, leagues []model.SportsLeague) error {
	log.Info("syncing scores")

	// Start sync log
	var syncLog *model.SportsSyncLog
	var err error
	if !*dryRun {
		syncLog, err = m.StartSportsSync(ctx, model.SportsSyncTypeScores, nil)
		if err != nil {
			log.WithError(err).Warn("failed to create sync log")
		}
	}

	// Finalize any stale events that fell out of the score update window
	if !*dryRun {
		staleCount, err := m.FinalizeStaleEvents(ctx)
		if err != nil {
			log.WithError(err).Warn("failed to finalize stale events")
		} else if staleCount > 0 {
			log.WithField("count", staleCount).Info("finalized stale events")
		}
	}

	// Find events that need score updates
	events, err := m.EventsNeedingScoreUpdate(ctx)
	if err != nil {
		if syncLog != nil {
			_ = syncLog.Complete(ctx, 0, false, err.Error())
		}
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

	log.WithField("count", totalEvents).Info("found events needing score update")

	updatedCount := 0
	for league, leagueEvents := range eventsByLeague {
		leagueLog := log.WithField("league", league)

		// Fetch today's and yesterday's scoreboards for this league
		now := time.Now()
		dates := []string{
			now.Format("20060102"),
			now.AddDate(0, 0, -1).Format("20060102"),
		}

		// Build lookup map by ESPN ID from both days
		espnEventMap := make(map[string]sports.Event)
		for _, date := range dates {
			scoreboardEvents, err := client.GetScoreboard(ctx, sports.League(league), sports.ScoreboardOptions{
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
				// Event not in scoreboard - fetch it individually via summary endpoint
				// This handles smaller school games that ESPN doesn't feature in daily scoreboards
				eventLog.Debug("event not found in scoreboard, fetching via summary endpoint")
				fetchedEvent, err := client.GetEventSummary(ctx, sports.League(league), dbEvent.ESPNID)
				if err != nil {
					eventLog.WithError(err).Debug("failed to fetch event summary")
					continue
				}
				espnEvent = *fetchedEvent
			}

			if err := processEvent(ctx, m, league, espnEvent); err != nil {
				eventLog.WithError(err).Error("failed to update event")
				continue
			}

			updatedCount++
		}
	}

	log.WithField("updatedCount", updatedCount).Info("finished syncing scores")

	if syncLog != nil {
		_ = syncLog.Complete(ctx, updatedCount, true, "")
	}

	return nil
}

func processEvent(ctx context.Context, m *model.Model, league model.SportsLeague, event sports.Event) error {
	if *dryRun {
		log.WithFields(logrus.Fields{
			"eventID":  event.ID,
			"homeTeam": event.HomeTeam.Abbreviation,
			"awayTeam": event.AwayTeam.Abbreviation,
			"status":   event.Status,
		}).Info("would process event")
		return nil
	}

	// Ensure teams exist
	homeTeam := m.NewSportsTeam()
	homeTeam.ID = event.HomeTeam.ID
	homeTeam.League = league
	homeTeam.Name = event.HomeTeam.Name
	homeTeam.FullName = event.HomeTeam.DisplayName
	homeTeam.Abbreviation = event.HomeTeam.Abbreviation
	if event.HomeTeam.Location != "" {
		homeTeam.Location = &event.HomeTeam.Location
	}
	if event.HomeTeam.Color != "" {
		homeTeam.Color = &event.HomeTeam.Color
	}
	if event.HomeTeam.AlternateColor != "" {
		homeTeam.AlternateColor = &event.HomeTeam.AlternateColor
	}
	if err := m.UpsertSportsTeam(ctx, nil, homeTeam); err != nil {
		return fmt.Errorf("upserting home team: %w", err)
	}

	awayTeam := m.NewSportsTeam()
	awayTeam.ID = event.AwayTeam.ID
	awayTeam.League = league
	awayTeam.Name = event.AwayTeam.Name
	awayTeam.FullName = event.AwayTeam.DisplayName
	awayTeam.Abbreviation = event.AwayTeam.Abbreviation
	if event.AwayTeam.Location != "" {
		awayTeam.Location = &event.AwayTeam.Location
	}
	if event.AwayTeam.Color != "" {
		awayTeam.Color = &event.AwayTeam.Color
	}
	if event.AwayTeam.AlternateColor != "" {
		awayTeam.AlternateColor = &event.AwayTeam.AlternateColor
	}
	if err := m.UpsertSportsTeam(ctx, nil, awayTeam); err != nil {
		return fmt.Errorf("upserting away team: %w", err)
	}

	// Load existing event to detect team ID changes
	existingEvent, err := m.SportsEventByESPNID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("loading existing event: %w", err)
	}

	// Create or update event
	sportsEvent := m.NewSportsEvent()
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

	sportsEvent.Period = &event.Period
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

	if err := m.UpsertSportsEvent(ctx, nil, sportsEvent); err != nil {
		return fmt.Errorf("upserting event: %w", err)
	}

	// Notify connected clients if score-relevant data changed
	if existingEvent != nil && sportsEventDataChanged(existingEvent, sportsEvent) {
		if err := m.NotifySportsEventUpdated(ctx, sportsEvent.ID); err != nil {
			log.WithError(err).WithField("eventID", sportsEvent.ID).Warn("failed to send sports_event_updated notification")
		}
	}

	// Sync grid team names and colors only when team IDs changed (or event is new)
	teamsChanged := existingEvent == nil ||
		existingEvent.HomeTeamID != event.HomeTeam.ID ||
		existingEvent.AwayTeamID != event.AwayTeam.ID
	if teamsChanged {
		gridCount, err := m.SyncGridsFromEvent(ctx, sportsEvent.ID)
		if err != nil {
			return fmt.Errorf("syncing grids from event: %w", err)
		}
		if gridCount > 0 {
			log.WithFields(logrus.Fields{
				"eventID":      sportsEvent.ID,
				"gridsUpdated": gridCount,
			}).Info("synced grid team names/colors from event")
		}
	}

	return nil
}

// sportsEventDataChanged returns true if any score-relevant field differs between two events.
func sportsEventDataChanged(existing, updated *model.SportsEvent) bool {
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
