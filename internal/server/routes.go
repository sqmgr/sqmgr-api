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
	"net/http"
	"path/filepath"
)

func (s *Server) setupRoutes() {
	s.Router.Use(s.middleware)

	// static directory
	s.Router.PathPrefix("/static/").Methods(http.MethodGet).Handler(http.StripPrefix("/static/", s.noDirListing(http.FileServer(http.Dir(filepath.Join("web", "static"))))))

	// static files
	s.Router.Path("/humans.txt").Methods(http.MethodGet).Handler(s.staticFileHandler(filepath.Join("web", "static", "humans.txt")))
	s.Router.Path("/robots.txt").Methods(http.MethodGet).Handler(s.staticFileHandler(filepath.Join("web", "static", "robots.txt")))

	// basic pags
	s.Router.Path("/").Methods(http.MethodGet).Handler(s.simpleGetHandler("index.html"))
	s.Router.Path("/about").Methods(http.MethodGet).Handler(s.simpleGetHandler("about.html"))
	s.Router.Path("/donate").Methods(http.MethodGet).Handler(s.simpleGetHandler("donate.html"))
	s.Router.Path("/privacy").Methods(http.MethodGet).Handler(s.simpleGetHandler("privacy.html"))
	s.Router.Path("/terms").Methods(http.MethodGet).Handler(s.simpleGetHandler("terms.html"))

	// login/logout functionality
	s.Router.Path("/login").Methods(http.MethodGet, http.MethodPost).Handler(s.loginHandler())
	s.Router.Path("/logout").Methods(http.MethodGet).Handler(s.loginHandler())

	// account management
	s.Router.Path("/account").Methods(http.MethodGet).Handler(s.authHandler(s.accountHandler()))
	s.Router.Path("/account/change-password").Methods(http.MethodGet, http.MethodPost).Handler(s.authHandler(s.accountChangePasswordHandler()))
	s.Router.Path("/account/delete").Methods(http.MethodGet, http.MethodPost).Handler(s.authHandler(s.accountDeleteHandler()))
	s.Router.Path("/account/deleted").Methods(http.MethodGet).Handler(s.authHandler(s.accountDeletedHandler()))
	s.Router.Path("/account/verify").Methods(http.MethodGet, http.MethodPost).Handler(s.authHandler(s.accountVerifyHandler()))

	// squares
	s.Router.Path("/squares/{token:[A-Za-z0-9_-]{6}}").Methods(http.MethodGet).Handler(s.squaresHandler())

	// signup
	s.Router.Path("/signup").Methods(http.MethodGet, http.MethodPost).Handler(s.signupHandler())
	s.Router.Path("/signup/complete").Methods(http.MethodGet).Handler(s.signupCompleteHandler())
	s.Router.Path("/signup/verify/{token:[A-Za-z0-9_-]{64}}").Methods(http.MethodGet).Handler(s.signupVerifyHandler())

	// square management
	s.Router.Path("/create").Methods(http.MethodGet, http.MethodPost).Handler(s.authHandler(s.createHandler()))

	// temporary
	s.Router.Path("/info").Methods(http.MethodGet).Handler(s.infoHandler())

	s.Router.NotFoundHandler = s.middleware(s.notFoundHandler())
}
