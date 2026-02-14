/*
Copyright (C) 2024 Tom Peters

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
	"testing"

	"github.com/onsi/gomega"
)

func TestIsValidNumberSetConfig(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Valid configs
	g.Expect(IsValidNumberSetConfig("standard")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetConfig("123f")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetConfig("1234f")).Should(gomega.BeFalse())
	g.Expect(IsValidNumberSetConfig("hf")).Should(gomega.BeTrue())

	// Invalid configs
	g.Expect(IsValidNumberSetConfig("")).Should(gomega.BeFalse())
	g.Expect(IsValidNumberSetConfig("invalid")).Should(gomega.BeFalse())
	g.Expect(IsValidNumberSetConfig("STANDARD")).Should(gomega.BeFalse())
}

func TestIsValidNumberSetType(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Valid types
	g.Expect(IsValidNumberSetType("all")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetType("q1")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetType("q2")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetType("q3")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetType("q4")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetType("half")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetType("final")).Should(gomega.BeTrue())

	// Invalid types
	g.Expect(IsValidNumberSetType("")).Should(gomega.BeFalse())
	g.Expect(IsValidNumberSetType("invalid")).Should(gomega.BeFalse())
	g.Expect(IsValidNumberSetType("Q1")).Should(gomega.BeFalse())
}

func TestGetSetTypes(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Standard config returns "all"
	setTypes := GetSetTypes(NumberSetConfigStandard)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeAll}))

	// q123f config returns 3 quarters + final
	setTypes = GetSetTypes(NumberSetConfig123F)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeQ1, NumberSetTypeQ2, NumberSetTypeQ3, NumberSetTypeFinal}))

	// hf config returns half + final
	setTypes = GetSetTypes(NumberSetConfigHF)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeHalf, NumberSetTypeFinal}))

	// Invalid config returns nil
	setTypes = GetSetTypes(NumberSetConfig("invalid"))
	g.Expect(setTypes).Should(gomega.BeNil())
}

func TestValidNumberSetConfigs(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	configs := ValidNumberSetConfigs()
	g.Expect(len(configs)).Should(gomega.Equal(3))

	// Check first config is "standard"
	g.Expect(configs[0].Key).Should(gomega.Equal(NumberSetConfigStandard))
	g.Expect(configs[0].Label).Should(gomega.Equal("Same"))
	g.Expect(configs[0].SetTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeAll}))
}

func TestNumberSetTypeInfos(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	infos := NumberSetTypeInfos()
	g.Expect(len(infos)).Should(gomega.Equal(7))

	// Check q1 info
	q1Info := infos[NumberSetTypeQ1]
	g.Expect(q1Info.Key).Should(gomega.Equal(NumberSetTypeQ1))
	g.Expect(q1Info.Label).Should(gomega.Equal("1st"))
	g.Expect(q1Info.LongLabel).Should(gomega.Equal("1st Quarter"))
}

func TestNumberSetTypeLongLabel(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Test all known types return correct long labels
	g.Expect(NumberSetTypeQ1.LongLabel()).Should(gomega.Equal("1st Quarter"))
	g.Expect(NumberSetTypeQ2.LongLabel()).Should(gomega.Equal("2nd Quarter"))
	g.Expect(NumberSetTypeQ3.LongLabel()).Should(gomega.Equal("3rd Quarter"))
	g.Expect(NumberSetTypeQ4.LongLabel()).Should(gomega.Equal("4th Quarter"))
	g.Expect(NumberSetTypeHalf.LongLabel()).Should(gomega.Equal("Halftime"))
	g.Expect(NumberSetTypeFinal.LongLabel()).Should(gomega.Equal("Final"))
	g.Expect(NumberSetTypeAll.LongLabel()).Should(gomega.Equal("Final"))

	// Unknown type returns the type string as fallback
	unknown := NumberSetType("unknown")
	g.Expect(unknown.LongLabel()).Should(gomega.Equal("unknown"))
}

func TestNumberSetConfigScan(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	var config NumberSetConfig

	// Scan from string
	err := config.Scan("1234")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(config).Should(gomega.Equal(NumberSetConfig1234))

	// Scan from bytes
	err = config.Scan([]byte("hf"))
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(config).Should(gomega.Equal(NumberSetConfigHF))

	// Scan from nil defaults to "standard"
	err = config.Scan(nil)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(config).Should(gomega.Equal(NumberSetConfigStandard))
}

func TestNumberSetTypeScan(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	var setType NumberSetType

	// Scan from string
	err := setType.Scan("q1")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(setType).Should(gomega.Equal(NumberSetTypeQ1))

	// Scan from bytes
	err = setType.Scan([]byte("final"))
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(setType).Should(gomega.Equal(NumberSetTypeFinal))

	// Scan from nil defaults to "all"
	err = setType.Scan(nil)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(setType).Should(gomega.Equal(NumberSetTypeAll))
}
