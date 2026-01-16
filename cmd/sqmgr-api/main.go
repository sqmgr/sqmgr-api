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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sqmgr/sqmgr-api/internal/config"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/database"
	"github.com/sqmgr/sqmgr-api/internal/server"

	_ "github.com/lib/pq"
)

var addr = flag.String("addr", getEnvOrElse("ADDR", ":5000"), "address for the server to listen on")
var sql = flag.String("sql", "./sql", "path to the SQL migrations")
var migrate = flag.Bool("migrate", false, "whether to run the database migrations")

const (
	readTimeout  = time.Second * 5
	writeTimeout = time.Second * 10
)

func main() {
	flag.Parse()

	setupLogger()

	if err := config.Load(); err != nil {
		logrus.WithError(err).Fatal("could not load config")
	}

	db, err := database.Open()
	if err != nil {
		logrus.WithError(err).Fatal("could not open database")
	}

	if *migrate {
		if err := database.ApplyMigrations(db, *sql); err != nil {
			logrus.WithError(err).Fatal("could not apply migrations")
		}
	}

	version := os.Getenv("SQMGR_VERSION")
	if version == "" {
		version = "dev"
	}

	s := server.New(version, db)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      handlers.ProxyHeaders(handlers.CombinedLoggingHandler(os.Stdout, s)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		logrus.WithField("addr", srv.Addr).Infof("listening")
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logrus.Fatal(err)
		}
	}()

	<-sig
	logrus.Infof("shutting down")
	if err := s.Shutdown(); err != nil {
		logrus.WithError(err).Errorln("could not shutdown server resources")
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.WithError(err).Fatalln("could not shut down server")
	}
	logrus.Infoln("shutdown complete")
}

func setupLogger() {
	if os.Getenv("LOG_LEVEL") != "" {
		lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
		if err != nil {
			logrus.WithError(err).Fatal("could not parse LOG_LEVEL")
		}

		logrus.SetLevel(lvl)
	}
}

func getEnvOrElse(key string, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return def
}
