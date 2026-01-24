/*
Copyright 2024 Tom Peters

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
	"testing"

	"github.com/onsi/gomega"
)

func TestIsValidNumberSetConfig(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Valid configs
	g.Expect(IsValidNumberSetConfig("standard")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetConfig("1234")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetConfig("123f")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetConfig("1234f")).Should(gomega.BeFalse())
	g.Expect(IsValidNumberSetConfig("hf")).Should(gomega.BeTrue())
	g.Expect(IsValidNumberSetConfig("h4")).Should(gomega.BeTrue())

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

	// q1234 config returns 4 quarters
	setTypes = GetSetTypes(NumberSetConfigQ1234)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeQ1, NumberSetTypeQ2, NumberSetTypeQ3, NumberSetTypeQ4}))

	// q123f config returns 3 quarters + final
	setTypes = GetSetTypes(NumberSetConfigQ123F)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeQ1, NumberSetTypeQ2, NumberSetTypeQ3, NumberSetTypeFinal}))

	// hf config returns half + final
	setTypes = GetSetTypes(NumberSetConfigHF)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeHalf, NumberSetTypeFinal}))

	// h4 config returns half + q4
	setTypes = GetSetTypes(NumberSetConfigH4)
	g.Expect(setTypes).Should(gomega.Equal([]NumberSetType{NumberSetTypeHalf, NumberSetTypeQ4}))

	// Invalid config returns nil
	setTypes = GetSetTypes(NumberSetConfig("invalid"))
	g.Expect(setTypes).Should(gomega.BeNil())
}

func TestValidNumberSetConfigs(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	configs := ValidNumberSetConfigs()
	g.Expect(len(configs)).Should(gomega.Equal(5))

	// Check first config is "standard"
	g.Expect(configs[0].Key).Should(gomega.Equal(NumberSetConfigStandard))
	g.Expect(configs[0].Label).Should(gomega.Equal("Standard"))
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
}

func TestNumberSetConfigScan(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	var config NumberSetConfig

	// Scan from string
	err := config.Scan("1234")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(config).Should(gomega.Equal(NumberSetConfigQ1234))

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
