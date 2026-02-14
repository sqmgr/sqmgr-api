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
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"net/http"
)

var optionsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func (s *Server) setupRoutes() {
	// Apply request ID, security headers, and rate limiting to all routes
	s.Router.Use(s.requestIDMiddleware)
	s.Router.Use(s.securityHeaders)
	s.Router.Use(s.rateLimiter.Limit)

	// these routes do NOT require auth
	s.Router.Path("/").Methods(http.MethodGet).Handler(s.getHealthEndpoint())
	s.Router.Path("/pool/configuration").Methods(http.MethodGet).Handler(s.getPoolConfiguration())
	s.Router.Path("/pool/{token:[A-Za-z0-9_-]+}/squares/public").Methods(http.MethodGet).Handler(s.getPoolTokenSquaresPublicEndpoint())
	s.Router.Path("/pool/{token:[A-Za-z0-9_-]+}/events").Methods(http.MethodGet).Handler(s.getPoolTokenEventsEndpoint())
	s.Router.Path("/user/guest").Methods(http.MethodPost).Handler(s.postUserGuestEndpoint())

	// Sports API routes (public, no auth required)
	s.Router.Path("/sports/leagues").Methods(http.MethodGet).Handler(s.getSportsLeaguesEndpoint())
	s.Router.Path("/sports/events").Methods(http.MethodGet).Handler(s.getSportsEventsEndpoint())
	s.Router.Path("/sports/events/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getSportsEventEndpoint())
	s.Router.Path("/sports/teams").Methods(http.MethodGet).Handler(s.getSportsTeamsEndpoint())

	// Deprecated BDL routes - alias to sports routes for backwards compatibility
	s.Router.Path("/bdl/leagues").Methods(http.MethodGet).Handler(s.getSportsLeaguesEndpoint())
	s.Router.Path("/bdl/events").Methods(http.MethodGet).Handler(s.getSportsEventsEndpoint())
	s.Router.Path("/bdl/events/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getSportsEventEndpoint())
	s.Router.Path("/bdl/teams").Methods(http.MethodGet).Handler(s.getSportsTeamsEndpoint())

	// these routes REQUIRE AUTH
	authRouter := s.NewRoute().Subrouter()
	authRouter.Use(s.authHandler)
	authRouter.Path("/pool").Methods(http.MethodPost).Handler(s.postPoolEndpoint())
	authRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/member").Methods(http.MethodPost).Handler(s.postPoolTokenMemberEndpoint())
	authRouter.Path("/user/self").Methods(http.MethodGet).Handler(s.getUserSelfEndpoint())
	authRouter.Path("/user/self/stats").Methods(http.MethodGet).Handler(s.getUserSelfStatsEndpoint())

	authPoolRouter := authRouter.NewRoute().Subrouter()
	authPoolRouter.Use(s.poolHandler)
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}").Methods(http.MethodGet).Handler(s.getPoolTokenEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}").Methods(http.MethodPost).Handler(s.postPoolTokenEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid").Methods(http.MethodGet).Handler(s.getPoolTokenGridEndpoint())

	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}").Methods(http.MethodDelete).Handler(s.deletePoolTokenGridIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getPoolTokenGridIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}").Methods(http.MethodPost).Handler(s.postPoolTokenGridIDEndpoint())

	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/invitetoken").Methods(http.MethodGet).Handler(s.getPoolTokenInviteTokenEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/log").Methods(http.MethodGet).Handler(s.getPoolTokenLogEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/square").Methods(http.MethodGet).Handler(s.getPoolTokenSquareEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/square/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getPoolTokenSquareIDEndpoint())
	authPoolRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/square/{id:[0-9]+}").Methods(http.MethodPost).Handler(s.postPoolTokenSquareIDEndpoint())

	authPoolGridRouter := authPoolRouter.NewRoute().Subrouter()
	authPoolGridRouter.Use(s.poolGridHandler)
	authPoolGridSquareAdminRouter := authPoolGridRouter.NewRoute().Subrouter()
	authPoolGridSquareAdminRouter.Use(s.poolGridSquareAdminHandler)
	authPoolGridSquareAdminRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}/square/{square_id:[0-9]+}/annotation").Methods(http.MethodPost).Handler(s.postPoolTokenGridIDSquareSquareIDAnnotationEndpoint())
	authPoolGridSquareAdminRouter.Path("/pool/{token:[A-Za-z0-9_-]+}/grid/{id:[0-9]+}/square/{square_id:[0-9]+}/annotation").Methods(http.MethodDelete).Handler(s.deletePoolTokenGridIDSquareSquareIDAnnotationEndpoint())

	authUserRouter := authRouter.NewRoute().Subrouter()
	authUserRouter.Use(s.userHandler)
	authUserRouter.Path("/user/{id:[0-9]+}/pool/{membership:(?:own|belong)}").Methods(http.MethodGet).Handler(s.getUserIDPoolMembershipEndpoint())
	authUserRouter.Path("/user/{id:[0-9]+}/pool/{token:[A-Za-z0-9_-]+}").Methods(http.MethodDelete).Handler(s.deleteUserIDPoolTokenEndpoint())
	authUserRouter.Path("/user/{id:[0-9]+}/guestjwt").Methods(http.MethodPost).Handler(s.postUserIDGuestJWT())

	// Admin routes - requires site admin privileges
	adminRouter := authRouter.NewRoute().Subrouter()
	adminRouter.Use(s.adminHandler)
	adminRouter.Path("/admin/stats").Methods(http.MethodGet).Handler(s.getAdminStatsEndpoint())
	adminRouter.Path("/admin/pools").Methods(http.MethodGet).Handler(s.getAdminPoolsEndpoint())
	adminRouter.Path("/admin/users").Methods(http.MethodGet).Handler(s.getAdminUsersEndpoint())
	adminRouter.Path("/admin/pool/{token:[A-Za-z0-9_-]+}/join").Methods(http.MethodPost).Handler(s.postAdminPoolJoinEndpoint())
	adminRouter.Path("/admin/user/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getAdminUserEndpoint())
	adminRouter.Path("/admin/user/{id:[0-9]+}/pools").Methods(http.MethodGet).Handler(s.getAdminUserPoolsEndpoint())
	adminRouter.Path("/admin/events").Methods(http.MethodGet).Handler(s.getAdminEventsEndpoint())
	adminRouter.Path("/admin/events/{id:[0-9]+}/grids").Methods(http.MethodGet).Handler(s.getAdminEventGridsEndpoint())

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
		AllowedMethods: []string{http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPatch},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	})
	s.Router.Use(c.Handler)
}
