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
