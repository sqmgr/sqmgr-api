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

package model

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestHashOAuthSecret(t *testing.T) {
	g := gomega.NewWithT(t)

	h := hashOAuthSecret("secret")
	g.Expect(h).Should(gomega.HaveLen(64))
	g.Expect(h).Should(gomega.Equal(hashOAuthSecret("secret")))
	g.Expect(h).ShouldNot(gomega.Equal(hashOAuthSecret("secret2")))
	g.Expect(h).ShouldNot(gomega.ContainSubstring("secret"))
}

func TestOAuthClientAllowsRedirectURI(t *testing.T) {
	g := gomega.NewWithT(t)

	c := &OAuthClient{RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback", "http://localhost:3000/cb"}}
	g.Expect(c.AllowsRedirectURI("https://claude.ai/api/mcp/auth_callback")).Should(gomega.BeTrue())
	g.Expect(c.AllowsRedirectURI("http://localhost:3000/cb")).Should(gomega.BeTrue())
	g.Expect(c.AllowsRedirectURI("https://claude.ai/api/mcp/auth_callback/")).Should(gomega.BeFalse(), "matching is exact")
	g.Expect(c.AllowsRedirectURI("https://evil.example.com")).Should(gomega.BeFalse())
	g.Expect(c.AllowsRedirectURI("")).Should(gomega.BeFalse())
}

func TestOAuthModelWithMock(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	g.Expect(err).Should(gomega.Succeed())
	defer db.Close()
	m := New(db)

	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	// Registering a client stores the name and redirect URIs and returns a generated ID
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO oauth_clients (id, name, redirect_uris) VALUES ($1, $2, $3) RETURNING created")).
		WithArgs(sqlmock.AnyArg(), "Claude", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created"}).AddRow(created))
	client, err := m.NewOAuthClient(ctx, "Claude", []string{"https://claude.ai/cb"})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(client.ID).Should(gomega.HaveLen(oauthClientIDLength))
	g.Expect(client.Name).Should(gomega.Equal("Claude"))
	g.Expect(client.RedirectURIs).Should(gomega.Equal([]string{"https://claude.ai/cb"}))
	g.Expect(client.Created).Should(gomega.Equal(created))

	// Unknown clients are reported distinctly
	mock.ExpectQuery("SELECT id, name, redirect_uris, created FROM oauth_clients").WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "redirect_uris", "created"}))
	_, err = m.OAuthClientByID(ctx, "missing")
	g.Expect(errors.Is(err, ErrOAuthClientNotFound)).Should(gomega.BeTrue())

	// Consuming a code deletes it; a missing code and an expired code are distinguished
	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WithArgs(hashOAuthSecret("nope")).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "user_id", "redirect_uri", "code_challenge", "scope", "expires_at", "expired"}))
	_, err = m.ConsumeOAuthAuthorizationCode(ctx, "nope")
	g.Expect(errors.Is(err, ErrOAuthGrantNotFound)).Should(gomega.BeTrue())

	codeRows := func(expired bool) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"client_id", "user_id", "redirect_uri", "code_challenge", "scope", "expires_at", "expired"}).
			AddRow("client1", int64(7), "https://claude.ai/cb", "challenge", "analytics:read", created, expired)
	}
	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WithArgs(hashOAuthSecret("old")).WillReturnRows(codeRows(true))
	_, err = m.ConsumeOAuthAuthorizationCode(ctx, "old")
	g.Expect(errors.Is(err, ErrOAuthGrantExpired)).Should(gomega.BeTrue())

	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WithArgs(hashOAuthSecret("good")).WillReturnRows(codeRows(false))
	grant, err := m.ConsumeOAuthAuthorizationCode(ctx, "good")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(*grant).Should(gomega.Equal(OAuthAuthorizationCode{
		ClientID: "client1", UserID: 7, RedirectURI: "https://claude.ai/cb", CodeChallenge: "challenge", Scope: "analytics:read", ExpiresAt: created,
	}))

	// Refresh tokens behave the same way
	mock.ExpectQuery("DELETE FROM oauth_refresh_tokens").WithArgs(hashOAuthSecret("stale")).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "user_id", "scope", "expires_at", "expired"}).AddRow("client1", int64(7), "analytics:read", created, true))
	_, err = m.ConsumeOAuthRefreshToken(ctx, "stale")
	g.Expect(errors.Is(err, ErrOAuthGrantExpired)).Should(gomega.BeTrue())

	mock.ExpectQuery("DELETE FROM oauth_refresh_tokens").WithArgs(hashOAuthSecret("fresh")).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "user_id", "scope", "expires_at", "expired"}).AddRow("client1", int64(7), "analytics:read", created, false))
	refresh, err := m.ConsumeOAuthRefreshToken(ctx, "fresh")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(refresh.UserID).Should(gomega.Equal(int64(7)))

	// Purging touches both tables
	mock.ExpectExec("DELETE FROM oauth_authorization_codes WHERE expires_at").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("DELETE FROM oauth_refresh_tokens WHERE expires_at").WillReturnResult(sqlmock.NewResult(0, 1))
	g.Expect(m.PurgeExpiredOAuthGrants(ctx)).Should(gomega.Succeed())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestOAuthIntegration(t *testing.T) {
	ensureIntegration(t)
	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|oauth-"+randString())
	g.Expect(err).Should(gomega.Succeed())

	client, err := m.NewOAuthClient(ctx, "Claude Desktop", []string{"https://claude.ai/api/mcp/auth_callback", "http://localhost:6274/oauth/callback"})
	g.Expect(err).Should(gomega.Succeed())

	loaded, err := m.OAuthClientByID(ctx, client.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(loaded.Name).Should(gomega.Equal("Claude Desktop"))
	g.Expect(loaded.RedirectURIs).Should(gomega.Equal(client.RedirectURIs))
	g.Expect(loaded.Created).ShouldNot(gomega.BeZero())

	// An authorization code can be consumed exactly once
	code, err := m.NewOAuthAuthorizationCode(ctx, OAuthAuthorizationCode{
		ClientID: client.ID, UserID: user.ID, RedirectURI: client.RedirectURIs[0], CodeChallenge: "challenge", Scope: "analytics:read",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(code).Should(gomega.HaveLen(oauthCodeLength))

	grant, err := m.ConsumeOAuthAuthorizationCode(ctx, code)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grant.ClientID).Should(gomega.Equal(client.ID))
	g.Expect(grant.UserID).Should(gomega.Equal(user.ID))
	g.Expect(grant.CodeChallenge).Should(gomega.Equal("challenge"))
	g.Expect(grant.ExpiresAt).Should(gomega.BeTemporally("~", time.Now().Add(10*time.Minute), time.Minute))

	_, err = m.ConsumeOAuthAuthorizationCode(ctx, code)
	g.Expect(errors.Is(err, ErrOAuthGrantNotFound)).Should(gomega.BeTrue(), "second use is rejected")

	// An expired code is rejected and removed
	expiredCode, err := m.NewOAuthAuthorizationCode(ctx, OAuthAuthorizationCode{
		ClientID: client.ID, UserID: user.ID, RedirectURI: client.RedirectURIs[0], CodeChallenge: "challenge", Scope: "analytics:read",
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.ConsumeOAuthAuthorizationCode(ctx, expiredCode)
	g.Expect(errors.Is(err, ErrOAuthGrantExpired)).Should(gomega.BeTrue())
	_, err = m.ConsumeOAuthAuthorizationCode(ctx, expiredCode)
	g.Expect(errors.Is(err, ErrOAuthGrantNotFound)).Should(gomega.BeTrue())

	// Refresh tokens rotate: consuming one removes it
	token, err := m.NewOAuthRefreshToken(ctx, OAuthRefreshToken{ClientID: client.ID, UserID: user.ID, Scope: "analytics:read", ExpiresAt: time.Now().Add(time.Hour)})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(token).Should(gomega.HaveLen(oauthRefreshTokenLength))

	refresh, err := m.ConsumeOAuthRefreshToken(ctx, token)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(refresh.UserID).Should(gomega.Equal(user.ID))
	g.Expect(refresh.Scope).Should(gomega.Equal("analytics:read"))
	_, err = m.ConsumeOAuthRefreshToken(ctx, token)
	g.Expect(errors.Is(err, ErrOAuthGrantNotFound)).Should(gomega.BeTrue())

	// Purging removes expired grants but keeps live ones
	_, err = m.NewOAuthRefreshToken(ctx, OAuthRefreshToken{ClientID: client.ID, UserID: user.ID, Scope: "analytics:read", ExpiresAt: time.Now().Add(-time.Hour)})
	g.Expect(err).Should(gomega.Succeed())
	live, err := m.NewOAuthRefreshToken(ctx, OAuthRefreshToken{ClientID: client.ID, UserID: user.ID, Scope: "analytics:read", ExpiresAt: time.Now().Add(time.Hour)})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(m.PurgeExpiredOAuthGrants(ctx)).Should(gomega.Succeed())

	var remaining int
	g.Expect(m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_refresh_tokens WHERE client_id = $1", client.ID).Scan(&remaining)).Should(gomega.Succeed())
	g.Expect(remaining).Should(gomega.Equal(1))
	_, err = m.ConsumeOAuthRefreshToken(ctx, live)
	g.Expect(err).Should(gomega.Succeed())
}
