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

// Command sqmgr-email-backfill stores Auth0 email addresses for users who
// have not logged in since emails started being saved at login. It is safe to
// re-run: only users with no stored email are looked up.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/internal/database"
	"github.com/sqmgr/sqmgr-api/pkg/auth0"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"golang.org/x/time/rate"
)

// maxConsecutiveFailures aborts the run when Auth0 keeps failing (bad
// credentials, sustained rate limiting) instead of logging every user.
const maxConsecutiveFailures = 10

var dryrun = flag.Bool("dry-run", false, "only output which users would be updated")
var rps = flag.Float64("rate", 2, "maximum Auth0 Management API requests per second")
var batchSize = flag.Int("batch-size", 100, "number of users to load from the database at a time")
var log = logrus.NewEntry(logrus.StandardLogger())

type emailFetcher interface {
	GetUserEmail(ctx context.Context, userID string) (string, error)
}

type result struct {
	Checked  int
	Updated  int
	NotFound int
	Failed   int
}

func main() {
	flag.Parse()

	if err := config.Load(); err != nil {
		log.WithError(err).Fatal("could not load config")
	}

	if *dryrun {
		log = log.WithField("dry-run", true)
	}

	client := auth0.NewClient(auth0.Config{
		Domain:       config.Auth0MgmtDomain(),
		ClientID:     config.Auth0MgmtClientID(),
		ClientSecret: config.Auth0MgmtClientSecret(),
	})
	if !client.IsConfigured() {
		log.Fatal("auth0 management client is not configured")
	}

	db, err := database.Open()
	if err != nil {
		log.WithError(err).Fatal("could not open database")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("starting")
	res, err := backfill(ctx, model.New(db), client, rate.NewLimiter(rate.Limit(*rps), 1), *batchSize, *dryrun)
	log = log.WithFields(logrus.Fields{
		"checked":  res.Checked,
		"updated":  res.Updated,
		"notFound": res.NotFound,
		"failed":   res.Failed,
	})
	if err != nil {
		log.WithError(err).Fatal("backfill stopped")
	}
	log.Info("finished")
}

// backfill looks up and stores the email for every Auth0 user without one.
// Users that fail or have no Auth0 email are skipped and left for a later run.
func backfill(ctx context.Context, m *model.Model, fetcher emailFetcher, limiter *rate.Limiter, batchSize int, dryRun bool) (result, error) {
	var res result
	var afterID int64
	consecutiveFailures := 0

	for {
		users, err := m.Auth0UsersWithoutEmail(ctx, afterID, batchSize)
		if err != nil {
			return res, err
		}
		if len(users) == 0 {
			return res, nil
		}

		for _, user := range users {
			afterID = user.ID
			userLog := log.WithFields(logrus.Fields{"userID": user.ID, "storeID": user.StoreID})

			if err := limiter.Wait(ctx); err != nil {
				return res, err
			}

			res.Checked++
			email, err := fetcher.GetUserEmail(ctx, user.StoreID)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return res, err
				}

				res.Failed++
				consecutiveFailures++
				userLog.WithError(err).Warn("could not get email from Auth0")
				if consecutiveFailures >= maxConsecutiveFailures {
					return res, fmt.Errorf("%d consecutive Auth0 failures, last: %w", consecutiveFailures, err)
				}
				continue
			}
			consecutiveFailures = 0

			if email == "" {
				res.NotFound++
				userLog.Info("no email in Auth0")
				continue
			}

			if dryRun {
				res.Updated++
				userLog.Info("would set email")
				continue
			}

			if err := user.SetEmail(ctx, email); err != nil {
				return res, err
			}
			res.Updated++
			userLog.Info("set email")
		}
	}
}
