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
	"testing"

	"github.com/lib/pq"
	"github.com/onsi/gomega"
)

func TestPGListenerHandleNotification_InvalidPayload(t *testing.T) {
	g := gomega.NewWithT(t)

	broker := NewPoolBroker()
	listener := &PGListener{
		broker: broker,
	}

	// Should not panic with invalid payload — just log and return
	g.Expect(func() {
		listener.handleNotification(context.Background(), &pq.Notification{
			Extra: "not-a-number",
		})
	}).ShouldNot(gomega.Panic())
}
