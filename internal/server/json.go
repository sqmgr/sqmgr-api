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
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr-api/internal/validator"
	"net/http"
)

const statusError = "error"

// ErrorResponse represents an error
type ErrorResponse struct {
	Status string `json:"status"`
	Error string `json:"error"`
	ValidationErrors validator.Errors `json:"validationErrors,omitempty"`
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	msg := ""
	if statusCode / 100 == 5 {
		msg = http.StatusText(statusCode)

		if err != nil {
			logrus.WithError(err).Error("an internal server error occurred")
		}
	} else if err == nil {
		msg = http.StatusText(statusCode)
	} else {
		msg = err.Error()
	}

	s.writeJSONResponse(w, statusCode, ErrorResponse{
		Status: statusError,
		Error:  msg,
	})
}

func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		logrus.WithError(err).Error("could not encode JSON")
	}
}

func (s *Server) parseJSONPayload(w http.ResponseWriter, r *http.Request, obj interface{}) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		s.writeErrorResponse(w, http.StatusUnsupportedMediaType, nil)
		return false
	}

	if err := json.NewDecoder(r.Body).Decode(obj)	; err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err)
		return false
	}

	return true
}
