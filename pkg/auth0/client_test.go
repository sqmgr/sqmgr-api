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

package auth0

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestNewClient(t *testing.T) {
	g := gomega.NewWithT(t)

	cfg := Config{
		Domain:       "test.auth0.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	client := NewClient(cfg)

	g.Expect(client).ShouldNot(gomega.BeNil())
	g.Expect(client.domain).Should(gomega.Equal("test.auth0.com"))
	g.Expect(client.clientID).Should(gomega.Equal("test-client-id"))
	g.Expect(client.clientSecret).Should(gomega.Equal("test-client-secret"))
}

func TestClient_IsConfigured(t *testing.T) {
	g := gomega.NewWithT(t)

	// Fully configured client
	fullClient := NewClient(Config{
		Domain:       "test.auth0.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	})
	g.Expect(fullClient.IsConfigured()).Should(gomega.BeTrue())

	// Missing domain
	noDomainClient := NewClient(Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	})
	g.Expect(noDomainClient.IsConfigured()).Should(gomega.BeFalse())

	// Missing client ID
	noClientIDClient := NewClient(Config{
		Domain:       "test.auth0.com",
		ClientSecret: "test-client-secret",
	})
	g.Expect(noClientIDClient.IsConfigured()).Should(gomega.BeFalse())

	// Missing client secret
	noSecretClient := NewClient(Config{
		Domain:   "test.auth0.com",
		ClientID: "test-client-id",
	})
	g.Expect(noSecretClient.IsConfigured()).Should(gomega.BeFalse())

	// Empty config
	emptyClient := NewClient(Config{})
	g.Expect(emptyClient.IsConfigured()).Should(gomega.BeFalse())
}
