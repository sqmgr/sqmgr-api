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

package database

import (
	"database/sql"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"

	// migrate requires this file
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Open will open a database based on an environment variable DSN
func Open() (*sql.DB, error) {
	db, err := sql.Open("postgres", config.DSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

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
