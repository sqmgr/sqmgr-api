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

package server

import (
	"context"
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
	model           *model.Model
	version         string
	keyLocker       *keylocker.KeyLocker
	smjwt           *smjwt.SMJWT
	rateLimiter     *RateLimiter
	authRateLimiter *RateLimiter
	auth0Client     *auth0.Client
	broker          *PoolBroker
	pgListener      *PGListener
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

	// Auth rate limit: 5 failed attempts per minute per IP
	authRL := NewRateLimiter(5.0/60.0, 5)

	// Initialize Auth0 Management API client
	auth0Client := auth0.NewClient(auth0.Config{
		Domain:       config.Auth0MgmtDomain(),
		ClientID:     config.Auth0MgmtClientID(),
		ClientSecret: config.Auth0MgmtClientSecret(),
	})

	s := &Server{
		Router:          mux.NewRouter(),
		model:           model.New(db),
		keyLocker:       keylocker.New(config.Auth0JWKSURL()),
		smjwt:           sj,
		version:         version,
		rateLimiter:     rl,
		authRateLimiter: authRL,
		auth0Client:     auth0Client,
		broker:          NewPoolBroker(),
	}

	s.setupRoutes()

	// Start PostgreSQL listener for real-time score update notifications
	pgListener, err := NewPGListener(config.DSN(), s.model, s.broker)
	if err != nil {
		logrus.WithError(err).Error("could not start pg listener for score updates (SSE notifications will not work)")
	} else {
		pgListener.Start(context.Background())
		s.pgListener = pgListener
	}

	return s
}

// Shutdown will handle any cleanup
func (s *Server) Shutdown() error {
	if s.pgListener != nil {
		if err := s.pgListener.Close(); err != nil {
			logrus.WithError(err).Error("could not close pg listener")
		}
	}
	return nil
}
