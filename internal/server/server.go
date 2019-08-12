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
	"github.com/weters/sqmgr-api/internal/config"
	"github.com/weters/sqmgr-api/internal/keylocker"
	"github.com/weters/sqmgr-api/pkg/model"
	"github.com/weters/sqmgr-api/pkg/smjwt"
)

// Server represents the SqMGR server
type Server struct {
	*mux.Router
	model     *model.Model
	version string
	keyLocker *keylocker.KeyLocker
	smjwt *smjwt.SMJWT
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

	s := &Server{
		Router:    mux.NewRouter(),
		model:     model.New(db),
		keyLocker: keylocker.New("https://sqmgr.auth0.com/.well-known/jwks.json"),
		smjwt:     sj,
		version:   version,
	}

	s.setupRoutes()

	return s
}

// Shutdown will handle any cleanup
func (s *Server) Shutdown() error {
	return nil
}
