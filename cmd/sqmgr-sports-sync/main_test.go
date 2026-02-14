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
	"testing"

	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

func intP(i int) *int       { return &i }
func strP(s string) *string { return &s }

func TestIntPtrEqual(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(intPtrEqual(nil, nil)).Should(gomega.BeTrue())
	g.Expect(intPtrEqual(intP(1), intP(1))).Should(gomega.BeTrue())
	g.Expect(intPtrEqual(intP(1), intP(2))).Should(gomega.BeFalse())
	g.Expect(intPtrEqual(nil, intP(1))).Should(gomega.BeFalse())
	g.Expect(intPtrEqual(intP(1), nil)).Should(gomega.BeFalse())
}

func TestStrPtrEqual(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(strPtrEqual(nil, nil)).Should(gomega.BeTrue())
	g.Expect(strPtrEqual(strP("a"), strP("a"))).Should(gomega.BeTrue())
	g.Expect(strPtrEqual(strP("a"), strP("b"))).Should(gomega.BeFalse())
	g.Expect(strPtrEqual(nil, strP("a"))).Should(gomega.BeFalse())
	g.Expect(strPtrEqual(strP("a"), nil)).Should(gomega.BeFalse())
}

func TestSportsEventDataChanged(t *testing.T) {
	g := gomega.NewWithT(t)

	base := func() *model.SportsEvent {
		return &model.SportsEvent{
			Status:    model.SportsEventStatusScheduled,
			HomeScore: intP(10),
			AwayScore: intP(7),
			Period:    intP(2),
			Clock:     strP("5:00"),
			HomeQ1:    intP(3),
			AwayQ1:    intP(7),
		}
	}

	t.Run("no change", func(t *testing.T) {
		g.Expect(sportsEventDataChanged(base(), base())).Should(gomega.BeFalse())
	})

	t.Run("status changed", func(t *testing.T) {
		updated := base()
		updated.Status = model.SportsEventStatusInProgress
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("home score changed", func(t *testing.T) {
		updated := base()
		updated.HomeScore = intP(14)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("away score changed", func(t *testing.T) {
		updated := base()
		updated.AwayScore = intP(14)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("period changed", func(t *testing.T) {
		updated := base()
		updated.Period = intP(3)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("clock changed", func(t *testing.T) {
		updated := base()
		updated.Clock = strP("4:30")
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("status detail changed", func(t *testing.T) {
		updated := base()
		updated.StatusDetail = strP("Halftime")
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("quarter scores changed", func(t *testing.T) {
		updated := base()
		updated.HomeQ1 = intP(7)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("Q2 changed", func(t *testing.T) {
		updated := base()
		updated.HomeQ2 = intP(10)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("Q3 changed", func(t *testing.T) {
		updated := base()
		updated.HomeQ3 = intP(5)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("Q4 changed", func(t *testing.T) {
		updated := base()
		updated.HomeQ4 = intP(5)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("OT changed", func(t *testing.T) {
		updated := base()
		updated.HomeOT = intP(3)
		g.Expect(sportsEventDataChanged(base(), updated)).Should(gomega.BeTrue())
	})

	t.Run("nil to non-nil", func(t *testing.T) {
		existing := &model.SportsEvent{Status: model.SportsEventStatusScheduled}
		updated := &model.SportsEvent{
			Status:    model.SportsEventStatusScheduled,
			HomeScore: intP(0),
		}
		g.Expect(sportsEventDataChanged(existing, updated)).Should(gomega.BeTrue())
	})

	t.Run("both nil scores", func(t *testing.T) {
		existing := &model.SportsEvent{Status: model.SportsEventStatusScheduled}
		updated := &model.SportsEvent{Status: model.SportsEventStatusScheduled}
		g.Expect(sportsEventDataChanged(existing, updated)).Should(gomega.BeFalse())
	})
}
