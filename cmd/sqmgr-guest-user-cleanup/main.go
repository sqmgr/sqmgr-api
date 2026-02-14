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
	"flag"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/internal/database"
)

var dryrun = flag.Bool("dry-run", false, "only output what will be deleted")
var log = logrus.NewEntry(logrus.StandardLogger())

func main() {
	flag.Parse()

	if err := config.Load(); err != nil {
		log.WithError(err).Fatal("could not load config")
	}

	if *dryrun {
		log = log.WithField("dry-run", true)
	}

	log.Info("starting")
	defer func() {
		log.Info("finished")
	}()

	db, err := database.Open()
	if err != nil {
		log.WithError(err).Fatal("could not open database")
	}

	deleteGuestUser, err := db.Prepare("DELETE FROM guest_users WHERE store = $1 AND store_id = $2")
	if err != nil {
		log.WithError(err).Fatal("could not prepare statement")
	}

	rows, err := db.Query(query)
	if err != nil {
		log.WithError(err).Fatal("could not query")
	}
	defer rows.Close()

	for rows.Next() {
		var store string
		var storeID string
		if err := rows.Scan(&store, &storeID); err != nil {
			log.WithError(err).Fatal("could not scan row")
		}

		log.WithFields(logrus.Fields{
			"store":   store,
			"storeID": storeID,
		}).Info("delete user")
		if !*dryrun {
			if _, err := deleteGuestUser.Exec(store, storeID); err != nil {
				log.WithError(err).Fatal("could not delete guest user")
			}
		}
	}
}

const query = `
WITH expired_users AS (
    SELECT gu.store, gu.store_id, u.id
    FROM guest_users gu
    LEFT JOIN users u ON gu.store = u.store AND gu.store_id = u.store_id
    WHERE gu.expires < NOW() AT TIME ZONE 'utc'
)
SELECT eu.store, eu.store_id
FROM expired_users eu
WHERE eu.id IS NULL OR (SELECT COUNT(*) FROM pool_squares WHERE user_id = eu.id) = 0`
