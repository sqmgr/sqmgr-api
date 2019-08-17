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
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr-api/internal/config"
	"github.com/weters/sqmgr-api/internal/database"
	_ "github.com/lib/pq"
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
			"store": store,
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
