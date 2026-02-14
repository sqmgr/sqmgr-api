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

package keylocker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var httpClient = &http.Client{Timeout: time.Second * 3}

// KeyLocker will keep track of Auth0 keys
type KeyLocker struct {
	url       string
	kidToCert map[interface{}]string
	mu        sync.RWMutex
	fetchMu   sync.Mutex
}

// New returns a new KeyLocker
func New(url string) *KeyLocker {
	return &KeyLocker{
		url:       url,
		kidToCert: make(map[interface{}]string),
	}
}

type jwks struct {
	Keys []struct {
		KTY string   `json:"kty"`
		KID string   `json:"kid"`
		Use string   `json:"use"`
		N   string   `json:"n"`
		E   string   `json:"e"`
		X5C []string `json:"x5c"`
	} `json:"keys"`
}

// GetPEMCert will return the PEM cert for the given JWT
func (k *KeyLocker) GetPEMCert(token *jwt.Token) (string, error) {
	k.mu.RLock()
	cert, ok := k.kidToCert[token.Header["kid"]]
	k.mu.RUnlock()

	if ok {
		return cert, nil
	}

	k.fetchMu.Lock()
	defer k.fetchMu.Unlock()

	resp, err := httpClient.Get(k.url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var jwksResp jwks
	if err := json.NewDecoder(resp.Body).Decode(&jwksResp); err != nil {
		return "", err
	}

	for _, key := range jwksResp.Keys {
		if key.KID == token.Header["kid"] {
			cert = fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----", key.X5C[0])
			break
		}
	}

	if cert == "" {
		return "", errors.New("unable to find appropriate key")
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	k.kidToCert[token.Header["kid"]] = cert

	return cert, nil
}
