package server

import (
	"fmt"
	"net/http"
)

func (s *Server) infoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.GetSession(w, r)
		user, err := session.LoggedInUser()
		session.Save()

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Session Values:")
		for key, value := range session.Values {
			fmt.Fprintf(w, "%s=%s\n", key, value)
		}

		fmt.Fprintln(w, "\nLogged in user:")

		if err != nil {
			fmt.Fprintf(w, "error getting logged in user: %v\n", err)
		} else {
			fmt.Fprintf(w, "email=%s\n", user.Email)
		}

		return
	}
}
