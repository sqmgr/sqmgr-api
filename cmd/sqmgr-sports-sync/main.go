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
	"os"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/internal/database"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/sports"
	"github.com/sqmgr/sqmgr-api/pkg/sportsync"
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

	log.Info("starting sports sync (ESPN)")
	defer func() {
		log.Info("finished sports sync")
	}()

	db, err := database.Open()
	if err != nil {
		log.WithError(err).Fatal("could not open database")
	}
	defer db.Close()

	if !*syncTeams && !*syncSchedule && !*syncScores {
		log.Fatal("must specify one of: --sync-teams, --sync-schedule, --sync-scores")
	}

	syncer := sportsync.New(model.New(db), sports.NewClient(sports.Config{Logger: log}), sportsync.Options{
		Logger: log,
		DryRun: *dryRun,
	})

	ctx := context.Background()
	leagues := getLeaguesToSync()

	if *syncTeams {
		if err := syncer.SyncTeams(ctx, leagues); err != nil {
			log.WithError(err).Fatal("failed to sync teams")
		}
	}

	if *syncSchedule {
		if err := syncer.SyncSchedule(ctx, leagues); err != nil {
			log.WithError(err).Fatal("failed to sync schedule")
		}
	}

	if *syncScores {
		if err := syncer.SyncScores(ctx, leagues); err != nil {
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
	return sportsync.AllLeagues()
}
