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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/weters/sqmgr/internal/server"
)

var addr = flag.String("addr", ":8080", "address for the server to listen on")

const (
	readTimeout  = time.Second * 5
	writeTimeout = time.Second * 10
)

func main() {
	flag.Parse()

	s := server.New()

	srv := &http.Server{
		Addr:         *addr,
		Handler:      handlers.CombinedLoggingHandler(os.Stdout, handlers.ProxyHeaders(s)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("Listening...")
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-sig
	log.Printf("Initiating shutdown...")
	if err := s.Shutdown(); err != nil {
		log.Printf("error shutting down server resources: %v", err)
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("error shutting down: %v", err)
	}
	log.Printf("Shutdown complete.")
}
