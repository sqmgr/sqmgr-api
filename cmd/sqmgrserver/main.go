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
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/server"

	_ "github.com/lib/pq"
)

var addr = flag.String("addr", ":8080", "address for the server to listen on")
var dev = flag.Bool("dev", false, "enabling dev mode turns on debug logging and template reloads")

const (
	readTimeout  = time.Second * 5
	writeTimeout = time.Second * 10
)

func main() {
	flag.Parse()

	db, err := openDB()
	if err != nil {
		logrus.Fatalf("could not open database: %v", err)
	}

	s := server.New(db)
	if *dev {
		logrus.Infof("enabling template reload")
		s.Reload = true

		logrus.SetLevel(logrus.DebugLevel)
	}

	if os.Getenv("LOG_LEVEL") != "" {
		lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
		if err != nil {
			logrus.WithError(err).Fatal("could not parse LOG_LEVEL")
		}

		logrus.SetLevel(lvl)
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      handlers.CombinedLoggingHandler(os.Stdout, handlers.ProxyHeaders(s)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	sig := make(chan os.Signal)
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

func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
