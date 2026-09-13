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
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/config"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// The API acts as a minimal OAuth 2.1 authorization server so that MCP clients
// (Claude Desktop, Claude Code, the MCP Inspector, ...) can connect to the
// admin analytics endpoint with a normal browser login instead of a hand-copied
// JWT. The flow is:
//
//  1. The client discovers metadata at /.well-known/oauth-protected-resource
//     and /.well-known/oauth-authorization-server.
//  2. It registers itself at POST /oauth/register (RFC 7591) and receives a
//     client_id. Clients are public; there is no client secret.
//  3. It opens the browser at the web app's /oauth/authorize page with a PKCE
//     challenge. The web app logs the user in through Auth0 as usual, checks
//     that they are a site admin, and calls POST /oauth/authorize here to mint
//     a single-use authorization code, then redirects to the client.
//  4. The client exchanges the code at POST /oauth/token for a short-lived
//     access token (a JWT signed with the SqMGR key, audience = the MCP
//     resource URL) and a rotating refresh token.

// OAuth lifetimes and limits.
const (
	oauthAccessTokenTTL   = time.Hour
	oauthRefreshTokenTTL  = 30 * 24 * time.Hour
	oauthCodeTTL          = 10 * time.Minute
	oauthMaxClientName    = 100
	oauthMaxRedirectURIs  = 10
	oauthMinVerifierLen   = 43
	oauthMaxVerifierLen   = 128
	oauthScopeAnalytics   = "analytics:read"
	oauthResourceName     = "SqMGR Admin Analytics"
	oauthMCPResourcePath  = "/admin/mcp"
	oauthAuthorizePagePth = "/oauth/authorize"
)

// oauthSupportedScopes lists every scope a client may request.
var oauthSupportedScopes = []string{oauthScopeAnalytics}

// pkceChallengePattern matches a base64url-encoded SHA-256 digest.
var pkceChallengePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// pkceVerifierPattern matches the unreserved characters RFC 7636 permits.
var pkceVerifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// mcpResourceURL is the identifier of the MCP endpoint: the audience of the
// access tokens issued for it.
func mcpResourceURL() string {
	return config.PublicURL() + oauthMCPResourcePath
}

// mcpResourceMetadataURL is where clients learn which authorization server
// protects the MCP endpoint (RFC 9728, path-suffixed form).
func mcpResourceMetadataURL() string {
	return config.PublicURL() + "/.well-known/oauth-protected-resource" + oauthMCPResourcePath
}

// oauthError is an RFC 6749 error response body.
type oauthError struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func (s *Server) writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Cache-Control", "no-store")
	s.writeJSONResponse(w, status, oauthError{Error: code, Description: description})
}

// getOAuthProtectedResourceMetadata serves RFC 9728 metadata for the MCP
// endpoint.
func (s *Server) getOAuthProtectedResourceMetadata() http.HandlerFunc {
	type response struct {
		Resource               string   `json:"resource"`
		AuthorizationServers   []string `json:"authorization_servers"`
		BearerMethodsSupported []string `json:"bearer_methods_supported"`
		ScopesSupported        []string `json:"scopes_supported"`
		ResourceName           string   `json:"resource_name"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		s.writeJSONResponse(w, http.StatusOK, response{
			Resource:               mcpResourceURL(),
			AuthorizationServers:   []string{config.PublicURL()},
			BearerMethodsSupported: []string{"header"},
			ScopesSupported:        oauthSupportedScopes,
			ResourceName:           oauthResourceName,
		})
	}
}

// getOAuthAuthorizationServerMetadata serves RFC 8414 metadata.
func (s *Server) getOAuthAuthorizationServerMetadata() http.HandlerFunc {
	type response struct {
		Issuer                            string   `json:"issuer"`
		AuthorizationEndpoint             string   `json:"authorization_endpoint"`
		TokenEndpoint                     string   `json:"token_endpoint"`
		RegistrationEndpoint              string   `json:"registration_endpoint"`
		ResponseTypesSupported            []string `json:"response_types_supported"`
		ResponseModesSupported            []string `json:"response_modes_supported"`
		GrantTypesSupported               []string `json:"grant_types_supported"`
		CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
		TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
		ScopesSupported                   []string `json:"scopes_supported"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		s.writeJSONResponse(w, http.StatusOK, response{
			Issuer:                            config.PublicURL(),
			AuthorizationEndpoint:             config.FrontendURL() + oauthAuthorizePagePth,
			TokenEndpoint:                     config.PublicURL() + "/oauth/token",
			RegistrationEndpoint:              config.PublicURL() + "/oauth/register",
			ResponseTypesSupported:            []string{"code"},
			ResponseModesSupported:            []string{"query"},
			GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
			CodeChallengeMethodsSupported:     []string{"S256"},
			TokenEndpointAuthMethodsSupported: []string{"none"},
			ScopesSupported:                   oauthSupportedScopes,
		})
	}
}

// validateRedirectURI accepts absolute https URLs, or http URLs on a loopback
// host for locally running clients. Fragments are never allowed.
func validateRedirectURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return errors.New("redirect_uri must be an absolute URL")
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return errors.New("redirect_uri must not contain a fragment")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return errors.New("http redirect URIs are only allowed on localhost")
	default:
		return errors.New("redirect_uri must use https (or http on localhost)")
	}
}

// sanitizeClientName trims the name, drops control characters, and bounds its
// length; an empty result falls back to a generic label.
func sanitizeClientName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	clean := b.String()
	if clean == "" {
		clean = "MCP client"
	}
	if len(clean) > oauthMaxClientName {
		clean = string([]rune(clean)[:oauthMaxClientName])
	}
	return clean
}

// postOAuthRegisterEndpoint implements dynamic client registration (RFC 7591).
func (s *Server) postOAuthRegisterEndpoint() http.HandlerFunc {
	type request struct {
		RedirectURIs            []string `json:"redirect_uris"`
		ClientName              string   `json:"client_name"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
	}
	type response struct {
		ClientID                string   `json:"client_id"`
		ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
		ClientName              string   `json:"client_name"`
		RedirectURIs            []string `json:"redirect_uris"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		Scope                   string   `json:"scope"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "request body must be JSON")
			return
		}

		if len(req.RedirectURIs) == 0 {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uris is required")
			return
		}
		if len(req.RedirectURIs) > oauthMaxRedirectURIs {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "too many redirect_uris")
			return
		}
		for _, uri := range req.RedirectURIs {
			if err := validateRedirectURI(uri); err != nil {
				s.writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", err.Error())
				return
			}
		}
		if req.TokenEndpointAuthMethod != "" && req.TokenEndpointAuthMethod != "none" {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "only public clients (token_endpoint_auth_method \"none\") are supported")
			return
		}
		for _, gt := range req.GrantTypes {
			if gt != "authorization_code" && gt != "refresh_token" {
				s.writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "unsupported grant_type "+strconv.Quote(gt))
				return
			}
		}
		for _, rt := range req.ResponseTypes {
			if rt != "code" {
				s.writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "unsupported response_type "+strconv.Quote(rt))
				return
			}
		}

		client, err := s.model.NewOAuthClient(r.Context(), sanitizeClientName(req.ClientName), req.RedirectURIs)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		logrus.WithFields(logrus.Fields{"clientID": client.ID, "clientName": client.Name}).Info("registered oauth client")

		w.Header().Set("Cache-Control", "no-store")
		s.writeJSONResponse(w, http.StatusCreated, response{
			ClientID:                client.ID,
			ClientIDIssuedAt:        client.Created.Unix(),
			ClientName:              client.Name,
			RedirectURIs:            client.RedirectURIs,
			TokenEndpointAuthMethod: "none",
			GrantTypes:              []string{"authorization_code", "refresh_token"},
			ResponseTypes:           []string{"code"},
			Scope:                   strings.Join(oauthSupportedScopes, " "),
		})
	}
}

// getOAuthClientEndpoint lets the authorization page show who is asking for
// access and confirm the redirect URI before the user approves.
func (s *Server) getOAuthClientEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, err := s.model.OAuthClientByID(r.Context(), mux.Vars(r)["id"])
		if err != nil {
			if errors.Is(err, model.ErrOAuthClientNotFound) {
				s.writeErrorResponse(w, http.StatusNotFound, errors.New("unknown client"))
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		s.writeJSONResponse(w, http.StatusOK, client)
	}
}

// parseScope validates a requested scope string and returns the granted
// scopes. An empty request grants every supported scope.
func parseScope(requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return strings.Join(oauthSupportedScopes, " "), nil
	}
	var granted []string
	for _, scope := range strings.Fields(requested) {
		supported := false
		for _, known := range oauthSupportedScopes {
			if scope == known {
				supported = true
				break
			}
		}
		if !supported {
			return "", errors.New("unsupported scope " + strconv.Quote(scope))
		}
		granted = append(granted, scope)
	}
	return strings.Join(granted, " "), nil
}

// resourceMatchesMCP reports whether a resource indicator (RFC 8707) names the
// MCP endpoint. An empty indicator is accepted since the server only protects
// one resource.
func resourceMatchesMCP(resource string) bool {
	if resource == "" {
		return true
	}
	return strings.TrimRight(resource, "/") == mcpResourceURL()
}

// redirectWithParams appends query parameters to a redirect URI, keeping any
// query it already carries.
func redirectWithParams(redirectURI string, params map[string]string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// postOAuthAuthorizeEndpoint is called by the web app once a site admin has
// approved or denied a client. It validates the authorization request,
// records the grant, and returns the URL the browser should be sent to.
func (s *Server) postOAuthAuthorizeEndpoint() http.HandlerFunc {
	type request struct {
		ClientID            string `json:"client_id"`
		RedirectURI         string `json:"redirect_uri"`
		ResponseType        string `json:"response_type"`
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
		State               string `json:"state"`
		Scope               string `json:"scope"`
		Resource            string `json:"resource"`
		Approve             bool   `json:"approve"`
	}
	type response struct {
		RedirectURI string `json:"redirectUri"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		var req request
		if !s.parseJSONPayload(w, r, &req) {
			return
		}

		// Errors in the client or redirect URI cannot be reported by redirect.
		client, err := s.model.OAuthClientByID(r.Context(), req.ClientID)
		if err != nil {
			if errors.Is(err, model.ErrOAuthClientNotFound) {
				s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "unknown client_id")
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		if !client.AllowsRedirectURI(req.RedirectURI) {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "redirect_uri is not registered for this client")
			return
		}

		// Everything else is reported to the client through the redirect.
		redirectError := func(code, description string) {
			s.writeJSONResponse(w, http.StatusOK, response{RedirectURI: redirectWithParams(req.RedirectURI, map[string]string{
				"error":             code,
				"error_description": description,
				"state":             req.State,
			})})
		}

		if !req.Approve {
			redirectError("access_denied", "the user declined the request")
			return
		}
		if req.ResponseType != "code" {
			redirectError("unsupported_response_type", "only the code response type is supported")
			return
		}
		if req.CodeChallengeMethod != "S256" || !pkceChallengePattern.MatchString(req.CodeChallenge) {
			redirectError("invalid_request", "a PKCE code_challenge with method S256 is required")
			return
		}
		if !resourceMatchesMCP(req.Resource) {
			redirectError("invalid_target", "unknown resource")
			return
		}
		scope, err := parseScope(req.Scope)
		if err != nil {
			redirectError("invalid_scope", err.Error())
			return
		}

		code, err := s.model.NewOAuthAuthorizationCode(r.Context(), model.OAuthAuthorizationCode{
			ClientID:      client.ID,
			UserID:        user.ID,
			RedirectURI:   req.RedirectURI,
			CodeChallenge: req.CodeChallenge,
			Scope:         scope,
			ExpiresAt:     time.Now().Add(oauthCodeTTL),
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		logrus.WithFields(logrus.Fields{"clientID": client.ID, "userID": user.ID}).Info("authorized oauth client")

		s.writeJSONResponse(w, http.StatusOK, response{RedirectURI: redirectWithParams(req.RedirectURI, map[string]string{
			"code":  code,
			"state": req.State,
		})})
	}
}

// isNotFound reports whether a model lookup found no row.
func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// verifyPKCE checks that verifier hashes to the stored S256 challenge.
func verifyPKCE(verifier, challenge string) bool {
	if len(verifier) < oauthMinVerifierLen || len(verifier) > oauthMaxVerifierLen || !pkceVerifierPattern.MatchString(verifier) {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// mcpTokenClaims are the claims of an access token issued for the MCP endpoint.
type mcpTokenClaims struct {
	jwt.RegisteredClaims
	Scope    string `json:"scope,omitempty"`
	ClientID string `json:"client_id,omitempty"`
}

// tokenResponse is the RFC 6749 token endpoint success body.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

// issueMCPTokens mints an access token for the MCP resource and a fresh
// refresh token for the user and client.
func (s *Server) issueMCPTokens(r *http.Request, userID int64, clientID, scope string) (*tokenResponse, error) {
	now := time.Now()
	accessToken, err := s.smjwt.Sign(mcpTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    model.IssuerSqMGR,
			Subject:   strconv.FormatInt(userID, 10),
			Audience:  jwt.ClaimStrings{mcpResourceURL()},
			ExpiresAt: jwt.NewNumericDate(now.Add(oauthAccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Scope:    scope,
		ClientID: clientID,
	})
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.model.NewOAuthRefreshToken(r.Context(), model.OAuthRefreshToken{
		ClientID:  clientID,
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: now.Add(oauthRefreshTokenTTL),
	})
	if err != nil {
		return nil, err
	}

	return &tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(oauthAccessTokenTTL.Seconds()),
		RefreshToken: refreshToken,
		Scope:        scope,
	}, nil
}

// postOAuthTokenEndpoint exchanges an authorization code or refresh token for
// tokens. Failed exchanges count against the auth rate limit.
func (s *Server) postOAuthTokenEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		if s.authRateLimiter.IsLimited(ip) {
			s.writeOAuthError(w, http.StatusTooManyRequests, "invalid_request", "too many failed attempts")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		if err := r.ParseForm(); err != nil {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "request body must be form encoded")
			return
		}

		fail := func(status int, code, description string) {
			s.authRateLimiter.RecordFailure(ip)
			s.writeOAuthError(w, status, code, description)
		}

		if !resourceMatchesMCP(r.PostFormValue("resource")) {
			fail(http.StatusBadRequest, "invalid_target", "unknown resource")
			return
		}

		var userID int64
		var clientID, scope string
		switch r.PostFormValue("grant_type") {
		case "authorization_code":
			code := r.PostFormValue("code")
			verifier := r.PostFormValue("code_verifier")
			if code == "" || verifier == "" || r.PostFormValue("client_id") == "" {
				fail(http.StatusBadRequest, "invalid_request", "code, code_verifier, and client_id are required")
				return
			}

			grant, err := s.model.ConsumeOAuthAuthorizationCode(r.Context(), code)
			if err != nil {
				if errors.Is(err, model.ErrOAuthGrantNotFound) || errors.Is(err, model.ErrOAuthGrantExpired) {
					fail(http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
					return
				}
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			if grant.ClientID != r.PostFormValue("client_id") || grant.RedirectURI != r.PostFormValue("redirect_uri") || !verifyPKCE(verifier, grant.CodeChallenge) {
				fail(http.StatusBadRequest, "invalid_grant", "authorization code does not match this client")
				return
			}
			userID, clientID, scope = grant.UserID, grant.ClientID, grant.Scope

		case "refresh_token":
			token := r.PostFormValue("refresh_token")
			if token == "" {
				fail(http.StatusBadRequest, "invalid_request", "refresh_token is required")
				return
			}

			grant, err := s.model.ConsumeOAuthRefreshToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, model.ErrOAuthGrantNotFound) || errors.Is(err, model.ErrOAuthGrantExpired) {
					fail(http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
					return
				}
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			if cid := r.PostFormValue("client_id"); cid != "" && cid != grant.ClientID {
				fail(http.StatusBadRequest, "invalid_grant", "refresh token does not belong to this client")
				return
			}
			if requested := r.PostFormValue("scope"); requested != "" && requested != grant.Scope {
				fail(http.StatusBadRequest, "invalid_scope", "scope cannot be changed on refresh")
				return
			}
			userID, clientID, scope = grant.UserID, grant.ClientID, grant.Scope

		case "":
			fail(http.StatusBadRequest, "invalid_request", "grant_type is required")
			return
		default:
			fail(http.StatusBadRequest, "unsupported_grant_type", "only authorization_code and refresh_token are supported")
			return
		}

		// The grant is only honored while the user is still a site admin.
		user, err := s.model.GetUserByID(r.Context(), userID)
		if err != nil || !user.IsSiteAdmin {
			if err != nil && !isNotFound(err) {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			fail(http.StatusBadRequest, "invalid_grant", "the user is no longer authorized")
			return
		}

		tokens, err := s.issueMCPTokens(r, user.ID, clientID, scope)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if err := s.model.PurgeExpiredOAuthGrants(r.Context()); err != nil {
			logrus.WithError(err).Warn("could not purge expired oauth grants")
		}

		logrus.WithFields(logrus.Fields{"clientID": clientID, "userID": user.ID, "grantType": r.PostFormValue("grant_type")}).Info("issued mcp tokens")

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		s.writeJSONResponse(w, http.StatusOK, tokens)
	}
}
