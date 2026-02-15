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
	"database/sql/driver"
	"fmt"
)

// NumberSetConfig represents the configuration for how number sets are organized in a pool
type NumberSetConfig string

const (
	// NumberSetConfigStandard means one set of numbers for all quarters (legacy behavior)
	NumberSetConfigStandard NumberSetConfig = "standard"
	// NumberSetConfig1234 means 1st, 2nd, 3rd, 4th quarter
	NumberSetConfig1234 NumberSetConfig = "1234"
	// NumberSetConfig123F means 1st, 2nd, 3rd, Final
	NumberSetConfig123F NumberSetConfig = "123f"
	// NumberSetConfigHF means Half, Final
	NumberSetConfigHF NumberSetConfig = "hf"
	// NumberSetConfigH4 means Half, 4th
	NumberSetConfigH4 NumberSetConfig = "h4"
)

// NumberSetType represents an individual number set identifier
type NumberSetType string

const (
	// NumberSetTypeAll is used when config is "standard" (legacy)
	NumberSetTypeAll NumberSetType = "all"
	// NumberSetTypeQ1 is for 1st quarter
	NumberSetTypeQ1 NumberSetType = "q1"
	// NumberSetTypeQ2 is for 2nd quarter
	NumberSetTypeQ2 NumberSetType = "q2"
	// NumberSetTypeQ3 is for 3rd quarter
	NumberSetTypeQ3 NumberSetType = "q3"
	// NumberSetTypeQ4 is for 4th quarter
	NumberSetTypeQ4 NumberSetType = "q4"
	// NumberSetTypeHalf is for halftime
	NumberSetTypeHalf NumberSetType = "half"
	// NumberSetTypeFinal is for final score
	NumberSetTypeFinal NumberSetType = "final"
)

// NumberSetConfigInfo contains metadata for a number set configuration
type NumberSetConfigInfo struct {
	Key      NumberSetConfig `json:"key"`
	Label    string          `json:"label"`
	SetTypes []NumberSetType `json:"setTypes"`
}

// NumberSetTypeInfo contains metadata for a number set type
type NumberSetTypeInfo struct {
	Key       NumberSetType `json:"key"`
	Label     string        `json:"label"`     // Short label: "1st", "Half"
	LongLabel string        `json:"longLabel"` // Long label: "1st Quarter", "Halftime"
}

// validNumberSetConfigs contains all valid configurations
var validNumberSetConfigs = []NumberSetConfigInfo{
	{
		Key:      NumberSetConfigStandard,
		Label:    "Same",
		SetTypes: []NumberSetType{NumberSetTypeAll},
	},
	{
		Key:      NumberSetConfig123F,
		Label:    "1st, 2nd, 3rd, Final",
		SetTypes: []NumberSetType{NumberSetTypeQ1, NumberSetTypeQ2, NumberSetTypeQ3, NumberSetTypeFinal},
	},
	{
		Key:      NumberSetConfigHF,
		Label:    "Half, Final",
		SetTypes: []NumberSetType{NumberSetTypeHalf, NumberSetTypeFinal},
	},
	/*
		{
			Key:      NumberSetConfigQ1234,
			Label:    "1st, 2nd, 3rd, 4th",
			SetTypes: []NumberSetType{NumberSetTypeQ1, NumberSetTypeQ2, NumberSetTypeQ3, NumberSetTypeQ4},
		},
		{
			Key:      NumberSetConfigH4,
			Label:    "Half, 4th",
			SetTypes: []NumberSetType{NumberSetTypeHalf, NumberSetTypeQ4},
		},
	*/
}

// numberSetTypeInfos contains metadata for all number set types
var numberSetTypeInfos = map[NumberSetType]NumberSetTypeInfo{
	NumberSetTypeAll:   {Key: NumberSetTypeAll, Label: "All", LongLabel: "Final"},
	NumberSetTypeQ1:    {Key: NumberSetTypeQ1, Label: "1st", LongLabel: "1st Quarter"},
	NumberSetTypeQ2:    {Key: NumberSetTypeQ2, Label: "2nd", LongLabel: "2nd Quarter"},
	NumberSetTypeQ3:    {Key: NumberSetTypeQ3, Label: "3rd", LongLabel: "3rd Quarter"},
	NumberSetTypeQ4:    {Key: NumberSetTypeQ4, Label: "4th", LongLabel: "4th Quarter"},
	NumberSetTypeHalf:  {Key: NumberSetTypeHalf, Label: "Half", LongLabel: "Halftime"},
	NumberSetTypeFinal: {Key: NumberSetTypeFinal, Label: "Final", LongLabel: "Final"},
}

// ValidNumberSetConfigs returns all valid number set configurations with metadata
func ValidNumberSetConfigs() []NumberSetConfigInfo {
	return validNumberSetConfigs
}

// NumberSetTypeInfos returns metadata for all number set types
func NumberSetTypeInfos() map[NumberSetType]NumberSetTypeInfo {
	return numberSetTypeInfos
}

// GetSetTypes returns the set types required for a given configuration
func GetSetTypes(config NumberSetConfig) []NumberSetType {
	for _, c := range validNumberSetConfigs {
		if c.Key == config {
			return c.SetTypes
		}
	}
	return nil
}

// IsValidNumberSetConfig returns true if the config is valid
func IsValidNumberSetConfig(config string) bool {
	for _, c := range validNumberSetConfigs {
		if string(c.Key) == config {
			return true
		}
	}
	return false
}

// IsValidNumberSetType returns true if the set type is valid
func IsValidNumberSetType(setType string) bool {
	_, ok := numberSetTypeInfos[NumberSetType(setType)]
	return ok
}

// IsValidNumberSetConfigForLeague returns true if the config is valid for the given sports league
func IsValidNumberSetConfigForLeague(config NumberSetConfig, league SportsLeague) bool {
	// NCAAB uses halves, not quarters, so "1st, 2nd, 3rd, Final" is not valid
	if league == SportsLeagueNCAAB && config == NumberSetConfig123F {
		return false
	}
	return IsValidNumberSetConfig(string(config))
}

// ValidNumberSetConfigsForLeague returns valid number set configurations for a specific league
func ValidNumberSetConfigsForLeague(league SportsLeague) []NumberSetConfigInfo {
	configs := make([]NumberSetConfigInfo, 0, len(validNumberSetConfigs))
	for _, c := range validNumberSetConfigs {
		if IsValidNumberSetConfigForLeague(c.Key, league) {
			configs = append(configs, c)
		}
	}
	return configs
}

// LongLabel returns the long descriptive label for a number set type
func (n NumberSetType) LongLabel() string {
	if info, ok := numberSetTypeInfos[n]; ok {
		return info.LongLabel
	}
	return string(n)
}

// Value implements driver.Valuer for database storage
func (n NumberSetConfig) Value() (driver.Value, error) {
	return string(n), nil
}

// Scan implements sql.Scanner for database retrieval
func (n *NumberSetConfig) Scan(value interface{}) error {
	if value == nil {
		*n = NumberSetConfigStandard
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*n = NumberSetConfig(v)
	case string:
		*n = NumberSetConfig(v)
	default:
		return fmt.Errorf("cannot scan %T into NumberSetConfig", value)
	}
	return nil
}

// Value implements driver.Valuer for database storage
func (n NumberSetType) Value() (driver.Value, error) {
	return string(n), nil
}

// Scan implements sql.Scanner for database retrieval
func (n *NumberSetType) Scan(value interface{}) error {
	if value == nil {
		*n = NumberSetTypeAll
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*n = NumberSetType(v)
	case string:
		*n = NumberSetType(v)
	default:
		return fmt.Errorf("cannot scan %T into NumberSetType", value)
	}
	return nil
}
