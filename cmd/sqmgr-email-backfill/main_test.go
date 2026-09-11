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

package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"golang.org/x/time/rate"
)

const selectUsersWithoutEmail = "SELECT .+ FROM users WHERE store = \\$1 AND \\(email IS NULL OR email = ''\\) AND id > \\$2"
const updateEmail = "UPDATE users SET email = \\$1 WHERE id = \\$2"

type fakeFetcher struct {
	emails map[string]string
	errs   map[string]error
	calls  []string
}

func (f *fakeFetcher) GetUserEmail(_ context.Context, userID string) (string, error) {
	f.calls = append(f.calls, userID)
	if err, ok := f.errs[userID]; ok {
		return "", err
	}
	return f.emails[userID], nil
}

func userRows(ids ...int64) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"id", "store", "store_id", "is_site_admin", "email", "created"})
	for _, id := range ids {
		rows.AddRow(id, model.UserStoreAuth0, fmt.Sprintf("auth0|%d", id), false, nil, time.Now())
	}
	return rows
}

func newMockModel(t *testing.T) (*model.Model, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return model.New(db), mock
}

func unlimited() *rate.Limiter {
	return rate.NewLimiter(rate.Inf, 1)
}

func TestBackfill(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	fetcher := &fakeFetcher{
		emails: map[string]string{
			"auth0|1": "one@example.com",
			"auth0|3": "three@example.com",
		},
		errs: map[string]error{
			"auth0|4": errors.New("user request failed with status 429"),
		},
	}

	// batch size of 2 forces keyset pagination across three queries
	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 2).
		WillReturnRows(userRows(1, 2))
	mock.ExpectExec(updateEmail).
		WithArgs("one@example.com", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(2), 2).
		WillReturnRows(userRows(3, 4))
	mock.ExpectExec(updateEmail).
		WithArgs("three@example.com", int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(4), 2).
		WillReturnRows(userRows())

	res, err := backfill(context.Background(), m, fetcher, unlimited(), 2, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res).Should(gomega.Equal(result{Checked: 4, Updated: 2, NotFound: 1, Failed: 1}))
	g.Expect(fetcher.calls).Should(gomega.Equal([]string{"auth0|1", "auth0|2", "auth0|3", "auth0|4"}))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestBackfill_DryRunDoesNotWrite(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	fetcher := &fakeFetcher{emails: map[string]string{"auth0|1": "one@example.com"}}

	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 10).
		WillReturnRows(userRows(1))
	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(1), 10).
		WillReturnRows(userRows())

	res, err := backfill(context.Background(), m, fetcher, unlimited(), 10, true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res).Should(gomega.Equal(result{Checked: 1, Updated: 1}))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestBackfill_AbortsAfterConsecutiveFailures(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	ids := make([]int64, maxConsecutiveFailures+5)
	errs := make(map[string]error, len(ids))
	for i := range ids {
		ids[i] = int64(i + 1)
		errs[fmt.Sprintf("auth0|%d", i+1)] = errors.New("token request failed with status 401")
	}
	fetcher := &fakeFetcher{errs: errs}

	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 100).
		WillReturnRows(userRows(ids...))

	res, err := backfill(context.Background(), m, fetcher, unlimited(), 100, false)
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("consecutive Auth0 failures")))
	g.Expect(res.Failed).Should(gomega.Equal(maxConsecutiveFailures))
	g.Expect(fetcher.calls).Should(gomega.HaveLen(maxConsecutiveFailures))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestBackfill_SuccessResetsFailureCount(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	// a success after every failure keeps the run going past the abort threshold
	ids := make([]int64, maxConsecutiveFailures*2)
	errs := make(map[string]error)
	emails := make(map[string]string)
	for i := range ids {
		ids[i] = int64(i + 1)
		storeID := fmt.Sprintf("auth0|%d", i+1)
		if i%2 == 0 {
			errs[storeID] = errors.New("user request failed with status 429")
		} else {
			emails[storeID] = fmt.Sprintf("user%d@example.com", i+1)
		}
	}
	fetcher := &fakeFetcher{errs: errs, emails: emails}

	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 100).
		WillReturnRows(userRows(ids...))
	for i := range ids {
		if i%2 == 1 {
			mock.ExpectExec(updateEmail).
				WithArgs(fmt.Sprintf("user%d@example.com", i+1), int64(i+1)).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}
	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, ids[len(ids)-1], 100).
		WillReturnRows(userRows())

	res, err := backfill(context.Background(), m, fetcher, unlimited(), 100, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res).Should(gomega.Equal(result{Checked: len(ids), Updated: maxConsecutiveFailures, Failed: maxConsecutiveFailures}))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestBackfill_QueryError(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 100).
		WillReturnError(errors.New("database unavailable"))

	_, err := backfill(context.Background(), m, &fakeFetcher{}, unlimited(), 100, false)
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("database unavailable")))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestBackfill_UpdateError(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	fetcher := &fakeFetcher{emails: map[string]string{"auth0|1": "one@example.com"}}

	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 100).
		WillReturnRows(userRows(1))
	mock.ExpectExec(updateEmail).
		WithArgs("one@example.com", int64(1)).
		WillReturnError(errors.New("read-only transaction"))

	res, err := backfill(context.Background(), m, fetcher, unlimited(), 100, false)
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("read-only transaction")))
	g.Expect(res.Updated).Should(gomega.Equal(0))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestBackfill_CanceledContextStops(t *testing.T) {
	g := gomega.NewWithT(t)
	m, mock := newMockModel(t)

	// an interrupted run surfaces as a wrapped context error from the client
	fetcher := &fakeFetcher{errs: map[string]error{"auth0|1": fmt.Errorf("requesting user: %w", context.Canceled)}}

	mock.ExpectQuery(selectUsersWithoutEmail).
		WithArgs(model.UserStoreAuth0, int64(0), 100).
		WillReturnRows(userRows(1, 2))

	res, err := backfill(context.Background(), m, fetcher, unlimited(), 100, false)
	g.Expect(err).Should(gomega.MatchError(context.Canceled))
	g.Expect(res.Failed).Should(gomega.Equal(0))
	g.Expect(fetcher.calls).Should(gomega.Equal([]string{"auth0|1"}))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
