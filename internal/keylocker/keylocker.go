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

package keylocker

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"sync"
	"time"
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
