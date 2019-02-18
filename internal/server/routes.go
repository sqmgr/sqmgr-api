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

import "net/http"

func (s *Server) setupRoutes() {
	s.Router.Methods(http.MethodGet).PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))
	s.Router.Methods(http.MethodGet).Path("/").Handler(s.simpleGetHandler("index.html"))
	s.Router.Methods(http.MethodGet).Path("/squares/{token:[A-Za-z0-9_-]+}").Handler(s.squaresGetHandler())
	s.Router.Methods(http.MethodGet, http.MethodPost).Path("/squares/{token:[A-Za-z0-9_-]+}/login").Handler(s.squaresLoginHandler())
	s.Router.Methods(http.MethodGet, http.MethodPost).Path("/create").Handler(s.createHandler())
	s.Router.Methods(http.MethodGet).Path("/donate").Handler(s.simpleGetHandler("donate.html"))

	s.Router.Methods(http.MethodPost).Path("/pwned").Handler(s.pwnedHandler())
}
