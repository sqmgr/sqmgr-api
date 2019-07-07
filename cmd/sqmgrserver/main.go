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
	"github.com/weters/sqmgr/internal/config"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/database"
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
	if err := config.Load(); err != nil {
		logrus.Fatalf("could not load config: %v", err)
	}

	db, err := database.Open()
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
		Handler:      noCacheHanlder(*dev, handlers.CombinedLoggingHandler(os.Stdout, handlers.ProxyHeaders(s))),
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

// noCacheHandler will set a "Cache-ControL: no-cache" header for all request when this is run with the -dev flag
func noCacheHanlder(inDev bool, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if inDev {
			w.Header().Set("Cache-Control", "no-cache")
		}

		next.ServeHTTP(w, r)
	}
}
