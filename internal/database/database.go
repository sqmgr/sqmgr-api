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

package database

import (
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"path/filepath"

	// migrate requires this file
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Open will open a database based on an environment variable DSN
func Open() (*sql.DB, error) {
	db, err := sql.Open("postgres", config.DSN())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// ApplyMigrations will apply the database migrations
func ApplyMigrations(db *sql.DB, dir string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	sourceURL := filepath.Join("file://", absDir)
	log := logrus.WithField("sourceURL", sourceURL)
	log.Info("apply migrations")
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return err
	}
	m.Log = &migrateLogger{log}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

type migrateLogger struct {
	*logrus.Entry
}

func (m migrateLogger) Printf(format string, v ...interface{}) {
	m.Entry.Infof(format, v...)
}

func (m migrateLogger) Verbose() bool {
	return false
}
