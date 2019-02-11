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

// Package validator validates user data
package validator

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/weters/sqmgr/internal/model"
)

var nonPrintableRx = regexp.MustCompile(`\p{C}`)

// Errors is a mapping of fields to a list of errors
type Errors map[string][]string

// Validator is the main object for validating user input
type Validator struct {
	Errors Errors
}

// New returns a new validator object
func New() *Validator {
	return &Validator{
		Errors: make(Errors),
	}
}

// Printable will ensure that all characters in the string can be printed to string (i.e. no control characters)
func (v *Validator) Printable(key, val string) string {
	if len(val) == 0 || nonPrintableRx.MatchString(val) {
		v.addError(key, "must be a valid string")
		return ""
	}

	return val
}

// Password will ensure that the confirmation matches the password and that they are a certain length
func (v *Validator) Password(key, pw, cpw string, minLen int) string {
	hasError := false
	if pw != cpw {
		v.addError(key, "passwords do not match")
		hasError = true
	}

	if len(pw) < minLen {
		v.addError(key, "password must be at least %d characters", minLen)
		hasError = true
	}

	if hasError {
		return ""
	}

	return pw
}

// Datetime will validate the provider datetime and return a a time.Time object in UTC.
// This will convert a timezoneOffset which is provided by JS like "-5" and convert it
// into a time zone that Go can understand like -0500 or +0000.
func (v *Validator) Datetime(key, datetime, timezoneOffset string) time.Time {
	tzInt, err := strconv.Atoi(timezoneOffset)
	if err != nil {
		log.Printf("invalid timezone found: %s", timezoneOffset)
	}

	tzInt *= 100
	tzStr := ""
	if tzInt < 0 {
		tzStr = fmt.Sprintf("%05d", tzInt)
	} else {
		tzStr = "+" + fmt.Sprintf("%04d", tzInt)
	}

	dt, err := time.Parse("2006-01-02T15:04-0700", datetime+tzStr)
	if err != nil {
		log.Printf("Got %s, err = %v", datetime+tzStr, err)
		v.addError(key, "must be a valid date and time")
		return time.Time{}
	}

	return dt.UTC()
}

// SquaresType will ensure that the string is a valid square type
func (v *Validator) SquaresType(key, val string) model.SquaresType {
	if !model.IsValidSquaresType(val) {
		v.addError(key, "must be a valid squares type")
		return model.SquaresType("")
	}

	return model.SquaresType(val)
}

// OK will return true if no errors were found
func (v *Validator) OK() bool {
	return len(v.Errors) == 0
}

func (v *Validator) addError(key string, format string, args ...interface{}) {
	slice, ok := v.Errors[key]
	if !ok {
		slice = make([]string, 0)
	}

	slice = append(slice, fmt.Sprintf(format, args...))
	v.Errors[key] = slice
}
