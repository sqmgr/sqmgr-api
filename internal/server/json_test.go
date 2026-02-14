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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestWriteErrorResponse_500WithContextCanceled_DoesNotLog(t *testing.T) {
	g := gomega.NewWithT(t)

	hook := test.NewGlobal()
	defer func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	}()

	s := &Server{broker: NewPoolBroker()}
	rec := httptest.NewRecorder()

	s.writeErrorResponse(rec, http.StatusInternalServerError, context.Canceled)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(hook.Entries).Should(gomega.BeEmpty())

	var body ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(body.Status).Should(gomega.Equal("error"))
	g.Expect(body.Error).Should(gomega.Equal("Internal Server Error"))
}

func TestWriteErrorResponse_500WithWrappedContextCanceled_DoesNotLog(t *testing.T) {
	g := gomega.NewWithT(t)

	hook := test.NewGlobal()
	defer func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	}()

	s := &Server{broker: NewPoolBroker()}
	rec := httptest.NewRecorder()

	wrappedErr := fmt.Errorf("scanning user row: %w", context.Canceled)
	s.writeErrorResponse(rec, http.StatusInternalServerError, wrappedErr)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(hook.Entries).Should(gomega.BeEmpty())
}

func TestWriteErrorResponse_500WithRealError_Logs(t *testing.T) {
	g := gomega.NewWithT(t)

	hook := test.NewGlobal()
	defer func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	}()

	s := &Server{broker: NewPoolBroker()}
	rec := httptest.NewRecorder()

	s.writeErrorResponse(rec, http.StatusInternalServerError, fmt.Errorf("something broke"))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(hook.Entries).Should(gomega.HaveLen(1))
	g.Expect(hook.Entries[0].Message).Should(gomega.Equal("an internal server error occurred"))
}

func TestWriteErrorResponse_500WithNilError_DoesNotLog(t *testing.T) {
	g := gomega.NewWithT(t)

	hook := test.NewGlobal()
	defer func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	}()

	s := &Server{broker: NewPoolBroker()}
	rec := httptest.NewRecorder()

	s.writeErrorResponse(rec, http.StatusInternalServerError, nil)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(hook.Entries).Should(gomega.BeEmpty())
}

func TestWriteErrorResponse_4xx_UsesErrorMessage(t *testing.T) {
	g := gomega.NewWithT(t)

	hook := test.NewGlobal()
	defer func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	}()

	s := &Server{broker: NewPoolBroker()}
	rec := httptest.NewRecorder()

	s.writeErrorResponse(rec, http.StatusBadRequest, fmt.Errorf("bad input"))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(hook.Entries).Should(gomega.BeEmpty())

	var body ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(body.Error).Should(gomega.Equal("bad input"))
}
