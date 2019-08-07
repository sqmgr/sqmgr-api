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
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"net/http"
)

var optionsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { })

func (s *Server) setupRoutes() {

	// these routes do NOT require auth
	s.Router.Path("/").Methods(http.MethodGet).Handler(s.getHealthEndpoint())
	s.Router.Path("/pool/configuration").Methods(http.MethodGet).Handler(s.getPoolConfiguration())
	s.Router.Path("/user/guest").Methods(http.MethodPost).Handler(s.postUserGuestEndpoint())

	// these routes REQUIRE AUTH
	authRouter := s.NewRoute().Subrouter()
	authRouter.Use(s.authHandler)
	authRouter.Path("/pool").Methods(http.MethodPost).Handler(s.postPoolEndpoint())
	authRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/member").Methods(http.MethodPost).Handler(s.postPoolTokenMemberEndpoint())
	authRouter.Path("/user/self").Methods(http.MethodGet).Handler(s.getUserSelfEndpoint())

	authPoolRouter := authRouter.NewRoute().Subrouter()
	authPoolRouter.Use(s.poolHandler)
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}").Methods(http.MethodGet).Handler(s.getPoolTokenEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/invitetoken").Methods(http.MethodGet).Handler(s.getPoolTokenInviteTokenEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}").Methods(http.MethodPost).Handler(s.postPoolTokenEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid").Methods(http.MethodGet).Handler(s.getPoolTokenGridEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}").Methods(http.MethodDelete).Handler(s.deletePoolTokenGridIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getPoolTokenGridIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}").Methods(http.MethodPost).Handler(s.postPoolTokenGridIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/log").Methods(http.MethodGet).Handler(s.getPoolTokenLogEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/square").Methods(http.MethodGet).Handler(s.getPoolTokenSquareEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/square/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getPoolTokenSquareIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/square/{id:[0-9]+}").Methods(http.MethodPost).Handler(s.postPoolTokenSquareIDEndpoint())

	authUserRouter := authRouter.NewRoute().Subrouter()
	authUserRouter.Use(s.userHandler)
	authUserRouter.Path("/user/{id:[0-9]+}/pool/{membership:(?:own|belong)}").Methods(http.MethodGet).Handler(s.getUserIDPoolMembershipEndpoint())
	authUserRouter.Path("/user/{id:[0-9]+}/pool/{token:[A-Za-z0-9_-]+}").Methods(http.MethodDelete).Handler(s.deleteUserIDPoolTokenEndpoint())
	authUserRouter.Path("/user/{id:[0-9]+}/guestjwt").Methods(http.MethodPost).Handler(s.postUserIDGuestJWT())

	pathTemplates := make(map[string]bool)


	// add OPTIONS route
	if err := s.Router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err == nil {
			pathTemplates[path] = true
		}

		return nil
	}); err != nil {
		logrus.WithError(err).Fatal("could not walk router")
	}

	for tpl := range pathTemplates {
		s.Router.Path(tpl).Methods(http.MethodOptions).Handler(optionsHandler)
	}


	c := cors.New(cors.Options{
		AllowedMethods:         []string{http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPatch},
		AllowedHeaders:         []string{"Authorization","Content-Type"},
	})
	s.Router.Use(c.Handler)
}
