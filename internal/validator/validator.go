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
	"net/mail"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/weters/pwned"
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
		v.AddError(key, "must be a valid string")
		return ""
	}

	return val
}

// Email will ensure the email address is valid
func (v *Validator) Email(key, email string) string {
	if _, err := mail.ParseAddress(email); err != nil {
		v.AddError(key, "must be a valid email address")
		return ""
	}

	return email
}

// NotPwnedPassword will ensure that the password provided has not been pwned.
func (v *Validator) NotPwnedPassword(key, pw string) string {
	count, err := pwned.Count(pw)
	if err != nil {
		// if we can't detect, just make a note of it, but don't fail
		logrus.WithError(err).Errorln("could not determined if password has been pwned")
	}

	if count > 0 {
		times := "times"
		if count == 1 {
			times = "time"
		}

		v.AddError(key, "the password you provided has been compromised at least %d %s. please use a different password", count, times)
		return ""
	}

	return pw
}

// Password will ensure that the confirmation matches the password and that they are a certain length
func (v *Validator) Password(key, pw, cpw string, minLen int) string {
	hasError := false
	if pw != cpw {
		v.AddError(key, "passwords do not match")
		hasError = true
	}

	if len(pw) < minLen {
		v.AddError(key, "password must be at least %d characters", minLen)
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
		logrus.Errorf("invalid timezone found: %s", timezoneOffset)
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
		logrus.WithError(err).Warnf("could not parse date string: %s", datetime+tzStr)
		v.AddError(key, "must be a valid date and time")
		return time.Time{}
	}

	return dt.UTC()
}

// SquaresType will ensure that the string is a valid square type
func (v *Validator) SquaresType(key, val string) model.SquaresType {
	if err := model.IsValidSquaresType(val); err != nil {
		v.AddError(key, "must be a valid squares type")
		return model.SquaresType("")
	}

	return model.SquaresType(val)
}

// OK will return true if no errors were found
func (v *Validator) OK() bool {
	return len(v.Errors) == 0
}

// AddError will add an error for the specified key
func (v *Validator) AddError(key string, format string, args ...interface{}) {
	slice, ok := v.Errors[key]
	if !ok {
		slice = make([]string, 0)
	}

	slice = append(slice, fmt.Sprintf(format, args...))
	v.Errors[key] = slice
}
