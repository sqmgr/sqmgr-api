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
