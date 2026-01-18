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

package server

import (
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/internal/keylocker"
	"github.com/sqmgr/sqmgr-api/pkg/auth0"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/smjwt"
)

// Server represents the SqMGR server
type Server struct {
	*mux.Router
	model       *model.Model
	version     string
	keyLocker   *keylocker.KeyLocker
	smjwt       *smjwt.SMJWT
	rateLimiter *RateLimiter
	auth0Client *auth0.Client
}

// New returns a new server object
func New(version string, db *sql.DB) *Server {
	sj := smjwt.New()
	if err := sj.LoadPublicKey(config.JWTPublicKey()); err != nil {
		logrus.WithError(err).Fatal("could not load public key")
	}
	if err := sj.LoadPrivateKey(config.JWTPrivateKey()); err != nil {
		logrus.WithError(err).Fatal("could not load private key")
	}

	// Rate limit: 10 requests per second with burst of 20
	rl := NewRateLimiter(10, 20)

	// Initialize Auth0 Management API client
	auth0Client := auth0.NewClient(auth0.Config{
		Domain:       config.Auth0MgmtDomain(),
		ClientID:     config.Auth0MgmtClientID(),
		ClientSecret: config.Auth0MgmtClientSecret(),
	})

	s := &Server{
		Router:      mux.NewRouter(),
		model:       model.New(db),
		keyLocker:   keylocker.New(config.Auth0JWKSURL()),
		smjwt:       sj,
		version:     version,
		rateLimiter: rl,
		auth0Client: auth0Client,
	}

	s.setupRoutes()

	return s
}

// Shutdown will handle any cleanup
func (s *Server) Shutdown() error {
	return nil
}
