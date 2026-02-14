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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/validator"
)

const statusError = "error"

// Error code constants for structured API responses
const (
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeRateLimited  = "RATE_LIMITED"
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeBadRequest   = "BAD_REQUEST"
)

// ErrorResponse represents an error
type ErrorResponse struct {
	Status           string           `json:"status"`
	Code             string           `json:"code,omitempty"`
	Error            string           `json:"error"`
	ValidationErrors validator.Errors `json:"validationErrors,omitempty"`
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	msg := ""
	code := ""
	if statusCode/100 == 5 {
		msg = http.StatusText(statusCode)
		code = ErrCodeInternal

		if err != nil && !errors.Is(err, context.Canceled) {
			logrus.WithError(err).Error("an internal server error occurred")
		}
	} else if err == nil {
		msg = http.StatusText(statusCode)
	} else {
		msg = err.Error()
	}

	// Set error code based on status code if not already set
	if code == "" {
		switch statusCode {
		case http.StatusNotFound:
			code = ErrCodeNotFound
		case http.StatusUnauthorized:
			code = ErrCodeUnauthorized
		case http.StatusForbidden:
			code = ErrCodeForbidden
		case http.StatusTooManyRequests:
			code = ErrCodeRateLimited
		case http.StatusBadRequest:
			code = ErrCodeBadRequest
		}
	}

	s.writeJSONResponse(w, statusCode, ErrorResponse{
		Status: statusError,
		Code:   code,
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

	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err)
		return false
	}

	return true
}
