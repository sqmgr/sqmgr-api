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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/smjwt"
)

const (
	testPublicURL   = "http://api.test:8000"
	testFrontendURL = "http://app.test:8080"
)

// newOAuthTestServer builds a Server with test signing keys, a loaded config,
// a mocked database, and the routes registered.
func newOAuthTestServer(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()

	t.Setenv("SQMGR_CONF_JWT_PUBLIC_KEY", "../../pkg/smjwt/testdata/public.pem")
	t.Setenv("SQMGR_CONF_JWT_PRIVATE_KEY", "../../pkg/smjwt/testdata/private.pem")
	t.Setenv("SQMGR_CONF_PUBLIC_URL", testPublicURL+"/")
	t.Setenv("SQMGR_CONF_FRONTEND_URL", testFrontendURL)
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	sj := smjwt.New()
	if err := sj.LoadPublicKey(config.JWTPublicKey()); err != nil {
		t.Fatal(err)
	}
	if err := sj.LoadPrivateKey(config.JWTPrivateKey()); err != nil {
		t.Fatal(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	s := &Server{
		Router:          mux.NewRouter(),
		model:           model.New(db),
		version:         "test",
		smjwt:           sj,
		rateLimiter:     NewRateLimiter(1000, 1000),
		authRateLimiter: NewRateLimiter(1000, 1000),
		broker:          NewPoolBroker(),
	}
	s.mcpServer = s.newMCPServer()
	s.setupRoutes()
	return s, mock
}

// pkcePair returns a verifier and its S256 challenge.
func pkcePair() (string, string) {
	verifier := strings.Repeat("v", 43)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestOAuthMetadataEndpoints(t *testing.T) {
	g := gomega.NewWithT(t)
	s, _ := newOAuthTestServer(t)

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	var as map[string]interface{}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &as)).Should(gomega.Succeed())
	g.Expect(as["issuer"]).Should(gomega.Equal(testPublicURL))
	g.Expect(as["authorization_endpoint"]).Should(gomega.Equal(testFrontendURL + "/oauth/authorize"))
	g.Expect(as["token_endpoint"]).Should(gomega.Equal(testPublicURL + "/oauth/token"))
	g.Expect(as["registration_endpoint"]).Should(gomega.Equal(testPublicURL + "/oauth/register"))
	g.Expect(as["code_challenge_methods_supported"]).Should(gomega.ConsistOf("S256"))
	g.Expect(as["grant_types_supported"]).Should(gomega.ConsistOf("authorization_code", "refresh_token"))
	g.Expect(as["token_endpoint_auth_methods_supported"]).Should(gomega.ConsistOf("none"))

	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/admin/mcp"} {
		rec = httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK), path)
		var pr map[string]interface{}
		g.Expect(json.Unmarshal(rec.Body.Bytes(), &pr)).Should(gomega.Succeed())
		g.Expect(pr["resource"]).Should(gomega.Equal(testPublicURL + "/admin/mcp"))
		g.Expect(pr["authorization_servers"]).Should(gomega.ConsistOf(testPublicURL))
		g.Expect(pr["scopes_supported"]).Should(gomega.ConsistOf("analytics:read"))
	}

	// An unauthenticated MCP request points clients at the resource metadata
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, mcpRequest(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(rec.Header().Get("WWW-Authenticate")).Should(gomega.Equal(`Bearer resource_metadata="` + testPublicURL + `/.well-known/oauth-protected-resource/admin/mcp"`))
}

func TestValidateRedirectURI(t *testing.T) {
	g := gomega.NewWithT(t)

	for _, ok := range []string{
		"https://claude.ai/api/mcp/auth_callback",
		"https://example.com/cb?x=1",
		"http://localhost:6274/oauth/callback",
		"http://127.0.0.1:8080/cb",
		"http://[::1]:9000/cb",
	} {
		g.Expect(validateRedirectURI(ok)).Should(gomega.Succeed(), ok)
	}

	for _, bad := range []string{
		"",
		"/relative",
		"not a url",
		"http://example.com/cb",
		"https://example.com/cb#fragment",
		"ftp://example.com/cb",
		"javascript:alert(1)",
	} {
		g.Expect(validateRedirectURI(bad)).Should(gomega.HaveOccurred(), bad)
	}
}

func TestOAuthHelpers(t *testing.T) {
	g := gomega.NewWithT(t)
	_, _ = newOAuthTestServer(t)

	g.Expect(sanitizeClientName("  Claude\x00 Desktop\n ")).Should(gomega.Equal("Claude Desktop"))
	g.Expect(sanitizeClientName("")).Should(gomega.Equal("MCP client"))
	g.Expect(sanitizeClientName(strings.Repeat("x", 500))).Should(gomega.HaveLen(oauthMaxClientName))

	scope, err := parseScope("")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(scope).Should(gomega.Equal("analytics:read"))
	scope, err = parseScope("  analytics:read ")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(scope).Should(gomega.Equal("analytics:read"))
	_, err = parseScope("analytics:read admin:write")
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("admin:write")))

	g.Expect(resourceMatchesMCP("")).Should(gomega.BeTrue())
	g.Expect(resourceMatchesMCP(testPublicURL + "/admin/mcp")).Should(gomega.BeTrue())
	g.Expect(resourceMatchesMCP(testPublicURL + "/admin/mcp/")).Should(gomega.BeTrue())
	g.Expect(resourceMatchesMCP("https://other.example.com/admin/mcp")).Should(gomega.BeFalse())

	redirect := redirectWithParams("https://claude.ai/cb?keep=1", map[string]string{"code": "abc", "state": "xyz", "empty": ""})
	u, err := url.Parse(redirect)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(u.Query().Get("keep")).Should(gomega.Equal("1"))
	g.Expect(u.Query().Get("code")).Should(gomega.Equal("abc"))
	g.Expect(u.Query().Get("state")).Should(gomega.Equal("xyz"))
	g.Expect(u.Query().Has("empty")).Should(gomega.BeFalse())

	verifier, challenge := pkcePair()
	g.Expect(verifyPKCE(verifier, challenge)).Should(gomega.BeTrue())
	g.Expect(verifyPKCE(verifier+"x", challenge)).Should(gomega.BeFalse())
	g.Expect(verifyPKCE("short", challenge)).Should(gomega.BeFalse())
	g.Expect(verifyPKCE(strings.Repeat("!", 43), challenge)).Should(gomega.BeFalse(), "invalid characters")
	g.Expect(verifyPKCE(strings.Repeat("v", 129), challenge)).Should(gomega.BeFalse(), "too long")

	g.Expect(matchAudience("api.sqmgr.com", []string{"api.sqmgr.com"})).Should(gomega.Equal("api.sqmgr.com"))
	g.Expect(matchAudience([]interface{}{"other", "api.sqmgr.com"}, []string{"mcp", "api.sqmgr.com"})).Should(gomega.Equal("api.sqmgr.com"))
	g.Expect(matchAudience([]interface{}{"mcp", "api.sqmgr.com"}, []string{"mcp", "api.sqmgr.com"})).Should(gomega.Equal("mcp"), "first accepted audience wins")
	g.Expect(matchAudience("nope", []string{"api.sqmgr.com"})).Should(gomega.BeEmpty())
	g.Expect(matchAudience(nil, []string{"api.sqmgr.com"})).Should(gomega.BeEmpty())
	g.Expect(matchAudience(42, []string{"api.sqmgr.com"})).Should(gomega.BeEmpty())
}

func registerRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestOAuthRegisterEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newOAuthTestServer(t)

	expectError := func(body, code string) {
		t.Helper()
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, registerRequest(t, body))
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest), body)
		var res oauthError
		g.Expect(json.Unmarshal(rec.Body.Bytes(), &res)).Should(gomega.Succeed())
		g.Expect(res.Error).Should(gomega.Equal(code), body)
	}

	expectError(`not json`, "invalid_client_metadata")
	expectError(`{}`, "invalid_redirect_uri")
	expectError(`{"redirect_uris":["http://example.com/cb"]}`, "invalid_redirect_uri")
	expectError(`{"redirect_uris":["https://example.com/cb"],"token_endpoint_auth_method":"client_secret_basic"}`, "invalid_client_metadata")
	expectError(`{"redirect_uris":["https://example.com/cb"],"grant_types":["implicit"]}`, "invalid_client_metadata")
	expectError(`{"redirect_uris":["https://example.com/cb"],"response_types":["token"]}`, "invalid_client_metadata")
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed(), "invalid registrations never reach the database")

	created := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("INSERT INTO oauth_clients").WithArgs(sqlmock.AnyArg(), "Claude", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"created"}).AddRow(created))

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, registerRequest(t, `{"client_name":"Claude","redirect_uris":["https://claude.ai/api/mcp/auth_callback"],"token_endpoint_auth_method":"none","grant_types":["authorization_code","refresh_token"],"response_types":["code"]}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusCreated), rec.Body.String())
	g.Expect(rec.Header().Get("Cache-Control")).Should(gomega.Equal("no-store"))

	var res map[string]interface{}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &res)).Should(gomega.Succeed())
	g.Expect(res["client_id"]).Should(gomega.HaveLen(32))
	g.Expect(res["client_name"]).Should(gomega.Equal("Claude"))
	g.Expect(res["client_id_issued_at"]).Should(gomega.BeEquivalentTo(created.Unix()))
	g.Expect(res["redirect_uris"]).Should(gomega.ConsistOf("https://claude.ai/api/mcp/auth_callback"))
	g.Expect(res["token_endpoint_auth_method"]).Should(gomega.Equal("none"))
	g.Expect(res).ShouldNot(gomega.HaveKey("client_secret"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

// clientRows returns a one-row result for OAuthClientByID.
func clientRows(id string, redirectURIs ...string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "name", "redirect_uris", "created"}).
		AddRow(id, "Claude", "{"+strings.Join(redirectURIs, ",")+"}", time.Now())
}

func TestOAuthClientEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newOAuthTestServer(t)

	mock.ExpectQuery("SELECT id, name, redirect_uris, created FROM oauth_clients").WithArgs("abc123").
		WillReturnRows(clientRows("abc123", "https://claude.ai/cb"))

	req := httptest.NewRequest(http.MethodGet, "/oauth/client/abc123", nil)
	rec := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), ctxUserKey, &model.User{ID: 5})
	s.getOAuthClientEndpoint().ServeHTTP(rec, mux.SetURLVars(req.WithContext(ctx), map[string]string{"id": "abc123"}))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	var res model.OAuthClient
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &res)).Should(gomega.Succeed())
	g.Expect(res.ID).Should(gomega.Equal("abc123"))
	g.Expect(res.Name).Should(gomega.Equal("Claude"))
	g.Expect(res.RedirectURIs).Should(gomega.Equal([]string{"https://claude.ai/cb"}))

	mock.ExpectQuery("SELECT id, name, redirect_uris, created FROM oauth_clients").WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "redirect_uris", "created"}))
	rec = httptest.NewRecorder()
	s.getOAuthClientEndpoint().ServeHTTP(rec, mux.SetURLVars(req.WithContext(ctx), map[string]string{"id": "missing"}))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))

	// The route itself requires authentication
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oauth/client/abc123", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func authorizeRequest(t *testing.T, user *model.User, body map[string]interface{}) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	return req.WithContext(context.WithValue(req.Context(), ctxUserKey, user))
}

func TestOAuthAuthorizeEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newOAuthTestServer(t)
	admin := &model.User{ID: 42, IsSiteAdmin: true}
	_, challenge := pkcePair()

	base := func() map[string]interface{} {
		return map[string]interface{}{
			"client_id":             "client1",
			"redirect_uri":          "https://claude.ai/cb",
			"response_type":         "code",
			"code_challenge":        challenge,
			"code_challenge_method": "S256",
			"state":                 "st4te",
			"scope":                 "analytics:read",
			"resource":              testPublicURL + "/admin/mcp",
			"approve":               true,
		}
	}

	expectClient := func() {
		mock.ExpectQuery("SELECT id, name, redirect_uris, created FROM oauth_clients").WithArgs("client1").
			WillReturnRows(clientRows("client1", "https://claude.ai/cb"))
	}

	call := func(body map[string]interface{}) (int, map[string]string) {
		t.Helper()
		rec := httptest.NewRecorder()
		s.postOAuthAuthorizeEndpoint().ServeHTTP(rec, authorizeRequest(t, admin, body))
		var res map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		return rec.Code, res
	}

	redirectQuery := func(res map[string]string) url.Values {
		t.Helper()
		u, err := url.Parse(res["redirectUri"])
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(u.Scheme + "://" + u.Host + u.Path).Should(gomega.Equal("https://claude.ai/cb"))
		return u.Query()
	}

	// Unknown client and unregistered redirect URI cannot be redirected
	mock.ExpectQuery("SELECT id, name, redirect_uris, created FROM oauth_clients").WithArgs("client1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "redirect_uris", "created"}))
	code, res := call(base())
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_request"))

	expectClient()
	body := base()
	body["redirect_uri"] = "https://evil.example.com/cb"
	code, res = call(body)
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_request"))

	// Denial and parameter problems are reported through the redirect
	expectClient()
	body = base()
	body["approve"] = false
	code, res = call(body)
	g.Expect(code).Should(gomega.Equal(http.StatusOK))
	q := redirectQuery(res)
	g.Expect(q.Get("error")).Should(gomega.Equal("access_denied"))
	g.Expect(q.Get("state")).Should(gomega.Equal("st4te"))
	g.Expect(q.Has("code")).Should(gomega.BeFalse())

	expectClient()
	body = base()
	body["code_challenge_method"] = "plain"
	_, res = call(body)
	g.Expect(redirectQuery(res).Get("error")).Should(gomega.Equal("invalid_request"))

	expectClient()
	body = base()
	body["response_type"] = "token"
	_, res = call(body)
	g.Expect(redirectQuery(res).Get("error")).Should(gomega.Equal("unsupported_response_type"))

	expectClient()
	body = base()
	body["scope"] = "admin:write"
	_, res = call(body)
	g.Expect(redirectQuery(res).Get("error")).Should(gomega.Equal("invalid_scope"))

	expectClient()
	body = base()
	body["resource"] = "https://other.example.com/mcp"
	_, res = call(body)
	g.Expect(redirectQuery(res).Get("error")).Should(gomega.Equal("invalid_target"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed(), "no code is stored for rejected requests")

	// Approval stores a code bound to the client, user, redirect URI, and challenge
	expectClient()
	mock.ExpectExec("INSERT INTO oauth_authorization_codes").
		WithArgs(sqlmock.AnyArg(), "client1", int64(42), "https://claude.ai/cb", challenge, "analytics:read", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	code, res = call(base())
	g.Expect(code).Should(gomega.Equal(http.StatusOK))
	q = redirectQuery(res)
	g.Expect(q.Get("code")).Should(gomega.HaveLen(48))
	g.Expect(q.Get("state")).Should(gomega.Equal("st4te"))
	g.Expect(q.Has("error")).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// The route requires a site admin
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader("{}")))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
}

func tokenRequest(t *testing.T, form url.Values) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func userRows(id int64, isSiteAdmin bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "store", "store_id", "is_site_admin", "email", "created"}).
		AddRow(id, "auth0", "auth0|abc", isSiteAdmin, "admin@example.com", time.Now())
}

func codeGrantRows(challenge string, expired bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"client_id", "user_id", "redirect_uri", "code_challenge", "scope", "expires_at", "expired"}).
		AddRow("client1", int64(42), "https://claude.ai/cb", challenge, "analytics:read", time.Now().Add(time.Minute), expired)
}

func expectTokenIssue(mock sqlmock.Sqlmock, userID int64, isSiteAdmin bool) {
	mock.ExpectQuery("SELECT id, store, store_id, is_site_admin, email, created FROM users WHERE id").WithArgs(userID).
		WillReturnRows(userRows(userID, isSiteAdmin))
	if !isSiteAdmin {
		return
	}
	mock.ExpectExec("INSERT INTO oauth_refresh_tokens").WithArgs(sqlmock.AnyArg(), "client1", userID, "analytics:read", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM oauth_authorization_codes WHERE expires_at").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM oauth_refresh_tokens WHERE expires_at").WillReturnResult(sqlmock.NewResult(0, 0))
}

func TestOAuthTokenEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newOAuthTestServer(t)
	verifier, challenge := pkcePair()

	call := func(form url.Values) (int, map[string]interface{}) {
		t.Helper()
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, tokenRequest(t, form))
		var res map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		return rec.Code, res
	}

	code, res := call(url.Values{})
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_request"))

	code, res = call(url.Values{"grant_type": {"client_credentials"}})
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("unsupported_grant_type"))

	code, res = call(url.Values{"grant_type": {"authorization_code"}, "code": {"abc"}})
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_request"))

	code, res = call(url.Values{"grant_type": {"authorization_code"}, "code": {"abc"}, "code_verifier": {verifier}, "client_id": {"client1"}, "resource": {"https://other.example.com"}})
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_target"))

	exchange := func(extra url.Values) url.Values {
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {"the-code"},
			"code_verifier": {verifier},
			"client_id":     {"client1"},
			"redirect_uri":  {"https://claude.ai/cb"},
			"resource":      {testPublicURL + "/admin/mcp"},
		}
		for k, v := range extra {
			form[k] = v
		}
		return form
	}

	// Unknown or expired codes
	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(codeGrantRows(challenge, false))
	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(sqlmock.NewRows([]string{"client_id", "user_id", "redirect_uri", "code_challenge", "scope", "expires_at", "expired"}))
	code, res = call(exchange(url.Values{"code_verifier": {strings.Repeat("w", 43)}}))
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"), "PKCE mismatch")
	code, res = call(exchange(nil))
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"), "unknown code")

	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(codeGrantRows(challenge, true))
	_, res = call(exchange(nil))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"), "expired code")

	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(codeGrantRows(challenge, false))
	_, res = call(exchange(url.Values{"client_id": {"other-client"}}))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"), "client mismatch")

	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(codeGrantRows(challenge, false))
	_, res = call(exchange(url.Values{"redirect_uri": {"https://claude.ai/other"}}))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"), "redirect mismatch")

	// A user who lost the site admin role cannot redeem their grant
	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(codeGrantRows(challenge, false))
	expectTokenIssue(mock, 42, false)
	code, res = call(exchange(nil))
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// A valid exchange issues an access token for the MCP resource and a refresh token
	mock.ExpectQuery("DELETE FROM oauth_authorization_codes").WillReturnRows(codeGrantRows(challenge, false))
	expectTokenIssue(mock, 42, true)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, tokenRequest(t, exchange(nil)))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK), rec.Body.String())
	g.Expect(rec.Header().Get("Cache-Control")).Should(gomega.Equal("no-store"))
	var tokens tokenResponse
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &tokens)).Should(gomega.Succeed())
	g.Expect(tokens.TokenType).Should(gomega.Equal("Bearer"))
	g.Expect(tokens.ExpiresIn).Should(gomega.Equal(int64(3600)))
	g.Expect(tokens.Scope).Should(gomega.Equal("analytics:read"))
	g.Expect(tokens.RefreshToken).Should(gomega.HaveLen(64))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	parsed, err := jwt.ParseWithClaims(tokens.AccessToken, &mcpTokenClaims{}, func(*jwt.Token) (interface{}, error) { return s.smjwt.PublicKey(), nil })
	g.Expect(err).Should(gomega.Succeed())
	claims := parsed.Claims.(*mcpTokenClaims)
	g.Expect(claims.Issuer).Should(gomega.Equal(model.IssuerSqMGR))
	g.Expect(claims.Subject).Should(gomega.Equal("42"))
	g.Expect(claims.Audience).Should(gomega.ConsistOf(testPublicURL + "/admin/mcp"))
	g.Expect(claims.Scope).Should(gomega.Equal("analytics:read"))
	g.Expect(claims.ClientID).Should(gomega.Equal("client1"))
	g.Expect(claims.ExpiresAt.Time).Should(gomega.BeTemporally("~", time.Now().Add(time.Hour), time.Minute))

	// The access token authenticates MCP requests as that user
	mock.ExpectQuery("SELECT id, store, store_id, is_site_admin, email, created FROM users WHERE id").WithArgs(int64(42)).WillReturnRows(userRows(42, true))
	var seen *model.User
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = userFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	s.mcpAuthHandler(next).ServeHTTP(rec, req)
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK), rec.Body.String())
	g.Expect(seen).ShouldNot(gomega.BeNil())
	g.Expect(seen.ID).Should(gomega.Equal(int64(42)))
	g.Expect(seen.IsSiteAdmin).Should(gomega.BeTrue())

	// ...but is rejected by the regular API auth, which only accepts the API audience
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/user/self", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	s.authHandler(next).ServeHTTP(rec, req)
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(rec.Header().Get("WWW-Authenticate")).Should(gomega.BeEmpty())

	// A token for a deleted user is unauthorized rather than an error
	mock.ExpectQuery("SELECT id, store, store_id, is_site_admin, email, created FROM users WHERE id").WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "store", "store_id", "is_site_admin", "email", "created"}))
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/admin/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	s.mcpAuthHandler(next).ServeHTTP(rec, req)
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(rec.Header().Get("WWW-Authenticate")).Should(gomega.ContainSubstring("resource_metadata"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// Refresh: rotation issues a new pair, client mismatch is rejected
	refreshRows := func(expired bool) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"client_id", "user_id", "scope", "expires_at", "expired"}).
			AddRow("client1", int64(42), "analytics:read", time.Now().Add(time.Hour), expired)
	}
	mock.ExpectQuery("DELETE FROM oauth_refresh_tokens").WithArgs(sqlmock.AnyArg()).WillReturnRows(refreshRows(false))
	expectTokenIssue(mock, 42, true)
	code, res = call(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.RefreshToken}, "client_id": {"client1"}})
	g.Expect(code).Should(gomega.Equal(http.StatusOK), res)
	g.Expect(res["refresh_token"]).ShouldNot(gomega.Equal(tokens.RefreshToken))
	g.Expect(res["access_token"]).ShouldNot(gomega.BeEmpty())

	mock.ExpectQuery("DELETE FROM oauth_refresh_tokens").WithArgs(sqlmock.AnyArg()).WillReturnRows(refreshRows(false))
	code, res = call(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"x"}, "client_id": {"other"}})
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"))

	mock.ExpectQuery("DELETE FROM oauth_refresh_tokens").WithArgs(sqlmock.AnyArg()).WillReturnRows(refreshRows(true))
	_, res = call(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"x"}})
	g.Expect(res["error"]).Should(gomega.Equal("invalid_grant"))

	code, res = call(url.Values{"grant_type": {"refresh_token"}})
	g.Expect(code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(res["error"]).Should(gomega.Equal("invalid_request"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestOAuthTokenEndpoint_RateLimitsFailures(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newOAuthTestServer(t)
	s.authRateLimiter = NewRateLimiter(1.0/60.0, 2)

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, tokenRequest(t, url.Values{"grant_type": {"bogus"}}))
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, tokenRequest(t, url.Values{"grant_type": {"bogus"}}))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusTooManyRequests))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestMCPAuthHandler_RejectsBadTokens(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newOAuthTestServer(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	expectUnauthorized := func(token string, why string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/mcp", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		s.mcpAuthHandler(next).ServeHTTP(rec, req)
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized), why)
		g.Expect(rec.Header().Get("WWW-Authenticate")).Should(gomega.ContainSubstring(`resource_metadata="`+testPublicURL), why)
	}

	sign := func(claims jwt.Claims) string {
		token, err := s.smjwt.Sign(claims)
		g.Expect(err).Should(gomega.Succeed())
		return token
	}
	now := time.Now()

	expectUnauthorized("", "missing token")
	expectUnauthorized("garbage", "malformed token")
	expectUnauthorized(sign(jwt.RegisteredClaims{Issuer: model.IssuerSqMGR, Subject: "42", Audience: jwt.ClaimStrings{"https://elsewhere.example.com/mcp"}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}), "wrong audience")
	expectUnauthorized(sign(jwt.RegisteredClaims{Issuer: model.IssuerSqMGR, Subject: "42", Audience: jwt.ClaimStrings{testPublicURL + "/admin/mcp"}, ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour))}), "expired")
	expectUnauthorized(sign(jwt.RegisteredClaims{Issuer: model.IssuerSqMGR, Subject: "not-a-number", Audience: jwt.ClaimStrings{testPublicURL + "/admin/mcp"}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}), "non-numeric subject")
	expectUnauthorized(sign(jwt.RegisteredClaims{Issuer: "https://unknown.example.com/", Subject: "42", Audience: jwt.ClaimStrings{testPublicURL + "/admin/mcp"}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}), "unknown issuer")
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed(), "rejected tokens never hit the database")

	// A guest JWT (API audience) still resolves through the regular path
	mock.ExpectQuery(regexp.QuoteMeta("FROM get_user($1, $2)")).WithArgs("sqmgr", "sqmgr|guest").
		WillReturnRows(sqlmock.NewRows([]string{"id", "store", "store_id", "is_site_admin", "email", "created"}).AddRow(int64(9), "sqmgr", "sqmgr|guest", false, nil, now))
	mock.ExpectQuery("guest_users").WithArgs("sqmgr", "sqmgr|guest").WillReturnRows(sqlmock.NewRows([]string{"expires"}).AddRow(now.Add(time.Hour)))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+sign(jwt.RegisteredClaims{Issuer: model.IssuerSqMGR, Subject: "sqmgr|guest", Audience: jwt.ClaimStrings{audienceSqMGR}, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}))
	s.mcpAuthHandler(s.adminHandler(next)).ServeHTTP(rec, req)
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden), "authenticated but not a site admin")
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
