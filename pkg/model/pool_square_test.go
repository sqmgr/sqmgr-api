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
	"github.com/onsi/gomega"
	"strings"
	"testing"
)

func TestPoolSquare_Claimant(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &PoolSquare{}

	okClaimant := strings.Repeat("é", 30)
	s.SetClaimant(okClaimant)
	g.Expect(s.claimant).Should(gomega.Equal(okClaimant))
	g.Expect(s.Claimant()).Should(gomega.Equal(okClaimant))

	tooLongClaimant := strings.Repeat("í", 31)
	s.SetClaimant(tooLongClaimant)
	g.Expect(s.claimant).ShouldNot(gomega.Equal(tooLongClaimant))
	g.Expect(s.Claimant()).ShouldNot(gomega.Equal(tooLongClaimant))
	g.Expect(s.Claimant()).Should(gomega.Equal(string([]rune(tooLongClaimant)[0:30])))
}

func TestSquareUserInfoJSON_RegisteredUser(t *testing.T) {
	g := gomega.NewWithT(t)

	userInfo := &SquareUserInfoJSON{
		UserType: "registered",
		Email:    "john@example.com",
	}

	g.Expect(userInfo.UserType).Should(gomega.Equal("registered"))
	g.Expect(userInfo.Email).Should(gomega.Equal("john@example.com"))
}

func TestSquareUserInfoJSON_GuestUser(t *testing.T) {
	g := gomega.NewWithT(t)

	userInfo := &SquareUserInfoJSON{
		UserType: "guest",
	}

	g.Expect(userInfo.UserType).Should(gomega.Equal("guest"))
	g.Expect(userInfo.Email).Should(gomega.BeEmpty())
}

func TestPoolSquareJSON_WithUserInfo(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &PoolSquare{
		SquareID: 42,
		State:    PoolSquareStateClaimed,
	}
	s.SetClaimant("John Doe")

	json := s.JSON()
	g.Expect(json.UserInfo).Should(gomega.BeNil())

	// Add user info
	json.UserInfo = &SquareUserInfoJSON{
		UserType: "registered",
		Email:    "john@example.com",
	}

	g.Expect(json.UserInfo).ShouldNot(gomega.BeNil())
	g.Expect(json.UserInfo.UserType).Should(gomega.Equal("registered"))
	g.Expect(json.UserInfo.Email).Should(gomega.Equal("john@example.com"))
}
