/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
		if err := doSyncScores(ctx, m, client); err != nil {
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

func doSyncScores(ctx context.Context, m *model.Model, client *sports.Client) error {
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

	// Find events that need score updates
	events, err := m.EventsNeedingScoreUpdate(ctx)
	if err != nil {
		if syncLog != nil {
			_ = syncLog.Complete(ctx, 0, false, err.Error())
		}
		return fmt.Errorf("querying events needing update: %w", err)
	}

	log.WithField("count", len(events)).Info("found events needing score update")

	// Group events by league for efficient API calls
	eventsByLeague := make(map[model.SportsLeague][]*model.SportsEvent)
	for _, event := range events {
		eventsByLeague[event.League] = append(eventsByLeague[event.League], event)
	}

	updatedCount := 0
	for league, leagueEvents := range eventsByLeague {
		leagueLog := log.WithField("league", league)

		// Fetch today's scoreboard for this league
		today := time.Now().Format("20060102")
		scoreboardEvents, err := client.GetScoreboard(ctx, sports.League(league), sports.ScoreboardOptions{
			Date: today,
		})
		if err != nil {
			leagueLog.WithError(err).Warn("failed to fetch scoreboard")
			continue
		}

		// Build lookup map by ESPN ID
		espnEventMap := make(map[string]sports.Event)
		for _, e := range scoreboardEvents {
			espnEventMap[e.ID] = e
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
				eventLog.Debug("event not found in today's scoreboard")
				continue
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

	// Create or update event
	sportsEvent := m.NewSportsEvent()
	sportsEvent.ESPNID = event.ID
	sportsEvent.League = league
	sportsEvent.HomeTeamID = event.HomeTeam.ID
	sportsEvent.AwayTeamID = event.AwayTeam.ID
	sportsEvent.EventDate = event.Date
	sportsEvent.Season = event.Season
	sportsEvent.Week = event.Week
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

	return nil
}
