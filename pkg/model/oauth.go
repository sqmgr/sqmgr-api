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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/sqmgr/sqmgr-api/pkg/tokengen"
)

// Errors returned by the OAuth model.
var (
	ErrOAuthClientNotFound = errors.New("model: oauth client not found")
	ErrOAuthGrantNotFound  = errors.New("model: oauth grant not found")
	ErrOAuthGrantExpired   = errors.New("model: oauth grant expired")
)

// Lengths of generated OAuth secrets.
const (
	oauthClientIDLength     = 32
	oauthCodeLength         = 48
	oauthRefreshTokenLength = 64
)

// OAuthClient is an application registered to request access on behalf of a
// user, such as an MCP client.
type OAuthClient struct {
	ID           string    `json:"clientId"`
	Name         string    `json:"clientName"`
	RedirectURIs []string  `json:"redirectUris"`
	Created      time.Time `json:"created"`
}

// AllowsRedirectURI reports whether uri exactly matches one of the client's
// registered redirect URIs.
func (c *OAuthClient) AllowsRedirectURI(uri string) bool {
	for _, registered := range c.RedirectURIs {
		if registered == uri {
			return true
		}
	}
	return false
}

// NewOAuthClient registers a client with a freshly generated client ID.
func (m *Model) NewOAuthClient(ctx context.Context, name string, redirectURIs []string) (*OAuthClient, error) {
	id, err := tokengen.Generate(oauthClientIDLength)
	if err != nil {
		return nil, fmt.Errorf("generating client id: %w", err)
	}

	c := &OAuthClient{ID: id, Name: name, RedirectURIs: redirectURIs}
	if err := m.DB.QueryRowContext(ctx,
		"INSERT INTO oauth_clients (id, name, redirect_uris) VALUES ($1, $2, $3) RETURNING created",
		c.ID, c.Name, pq.Array(c.RedirectURIs),
	).Scan(&c.Created); err != nil {
		return nil, fmt.Errorf("inserting oauth client: %w", err)
	}

	return c, nil
}

// OAuthClientByID looks a client up by its client ID.
func (m *Model) OAuthClientByID(ctx context.Context, id string) (*OAuthClient, error) {
	c := &OAuthClient{}
	err := m.DB.QueryRowContext(ctx,
		"SELECT id, name, redirect_uris, created FROM oauth_clients WHERE id = $1", id,
	).Scan(&c.ID, &c.Name, pq.Array(&c.RedirectURIs), &c.Created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthClientNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying oauth client: %w", err)
	}
	return c, nil
}

// OAuthAuthorizationCode is a pending authorization grant. Only a hash of the
// code itself is stored.
type OAuthAuthorizationCode struct {
	ClientID      string
	UserID        int64
	RedirectURI   string
	CodeChallenge string
	Scope         string
	ExpiresAt     time.Time
}

// NewOAuthAuthorizationCode stores the grant and returns the plaintext code to
// hand to the client. ExpiresAt must be set by the caller.
func (m *Model) NewOAuthAuthorizationCode(ctx context.Context, grant OAuthAuthorizationCode) (string, error) {
	code, err := tokengen.Generate(oauthCodeLength)
	if err != nil {
		return "", fmt.Errorf("generating authorization code: %w", err)
	}

	if _, err := m.DB.ExecContext(ctx, `
		INSERT INTO oauth_authorization_codes (code_hash, client_id, user_id, redirect_uri, code_challenge, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		hashOAuthSecret(code), grant.ClientID, grant.UserID, grant.RedirectURI, grant.CodeChallenge, grant.Scope, grant.ExpiresAt.UTC(),
	); err != nil {
		return "", fmt.Errorf("inserting authorization code: %w", err)
	}

	return code, nil
}

// ConsumeOAuthAuthorizationCode deletes the grant for code and returns it, so
// a code can be exchanged at most once. An expired grant is still deleted but
// reported as ErrOAuthGrantExpired.
func (m *Model) ConsumeOAuthAuthorizationCode(ctx context.Context, code string) (*OAuthAuthorizationCode, error) {
	grant := &OAuthAuthorizationCode{}
	var expired bool
	err := m.DB.QueryRowContext(ctx, `
		DELETE FROM oauth_authorization_codes
		WHERE code_hash = $1
		RETURNING client_id, user_id, redirect_uri, code_challenge, scope, expires_at, expires_at <= (NOW() AT TIME ZONE 'utc')`,
		hashOAuthSecret(code),
	).Scan(&grant.ClientID, &grant.UserID, &grant.RedirectURI, &grant.CodeChallenge, &grant.Scope, &grant.ExpiresAt, &expired)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthGrantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consuming authorization code: %w", err)
	}
	if expired {
		return nil, ErrOAuthGrantExpired
	}
	return grant, nil
}

// OAuthRefreshToken is a long-lived grant that can be exchanged for a new
// access token. Only a hash of the token itself is stored.
type OAuthRefreshToken struct {
	ClientID  string
	UserID    int64
	Scope     string
	ExpiresAt time.Time
}

// NewOAuthRefreshToken stores the grant and returns the plaintext token.
func (m *Model) NewOAuthRefreshToken(ctx context.Context, grant OAuthRefreshToken) (string, error) {
	token, err := tokengen.Generate(oauthRefreshTokenLength)
	if err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}

	if _, err := m.DB.ExecContext(ctx, `
		INSERT INTO oauth_refresh_tokens (token_hash, client_id, user_id, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		hashOAuthSecret(token), grant.ClientID, grant.UserID, grant.Scope, grant.ExpiresAt.UTC(),
	); err != nil {
		return "", fmt.Errorf("inserting refresh token: %w", err)
	}

	return token, nil
}

// ConsumeOAuthRefreshToken deletes the grant for token and returns it, which
// rotates the token: the caller is expected to issue a replacement.
func (m *Model) ConsumeOAuthRefreshToken(ctx context.Context, token string) (*OAuthRefreshToken, error) {
	grant := &OAuthRefreshToken{}
	var expired bool
	err := m.DB.QueryRowContext(ctx, `
		DELETE FROM oauth_refresh_tokens
		WHERE token_hash = $1
		RETURNING client_id, user_id, scope, expires_at, expires_at <= (NOW() AT TIME ZONE 'utc')`,
		hashOAuthSecret(token),
	).Scan(&grant.ClientID, &grant.UserID, &grant.Scope, &grant.ExpiresAt, &expired)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthGrantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consuming refresh token: %w", err)
	}
	if expired {
		return nil, ErrOAuthGrantExpired
	}
	return grant, nil
}

// PurgeExpiredOAuthGrants removes authorization codes and refresh tokens that
// can no longer be used.
func (m *Model) PurgeExpiredOAuthGrants(ctx context.Context) error {
	for _, table := range []string{"oauth_authorization_codes", "oauth_refresh_tokens"} {
		if _, err := m.DB.ExecContext(ctx, "DELETE FROM "+table+" WHERE expires_at <= (NOW() AT TIME ZONE 'utc')"); err != nil {
			return fmt.Errorf("purging %s: %w", table, err)
		}
	}
	return nil
}

// hashOAuthSecret returns the hex SHA-256 digest used to store codes and
// refresh tokens without keeping the secret itself.
func hashOAuthSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
