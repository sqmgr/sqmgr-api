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
	"math"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/weters/pwned"
	"github.com/weters/sqmgr/internal/model"
)

var nonPrintableRx = regexp.MustCompile(`\p{C}`)

// \r\n (\x0d \x0a) is included in \p{C} (specifically \p{Cc}, so we need to work around it
var nonPrintableExcludeNewlineRx = regexp.MustCompile(`[\p{Cf}\p{Co}\p{Cs}\x00-\x09\x0b\x0c\x0e-\x1f\x7f-\x9f]`)
var colorRx = regexp.MustCompile(`^#[a-fA-F0-9]{6}\z`)
var wordCharRx = regexp.MustCompile(`[\p{L}\p{N}]`)

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

// MaxLength will ensure that the string is at most maxLength characters. 0 is valid.
func (v *Validator) MaxLength(key, val string, maxLength int) string {
	if utf8.RuneCountInString(val) > maxLength {
		v.AddError(key, "must be <= %d characters", maxLength)
		return ""
	}

	return val
}

// InverseRegexp will make sure that the string is non-empty and does not match the regex. It can be empty if isOptional is true.
func (v *Validator) InverseRegexp(key, val string, rx *regexp.Regexp, isOptional ...bool) string {
	if len(isOptional) > 0 && isOptional[0] && len(val) == 0 {
		return ""
	}

	if len(val) == 0 || rx.MatchString(val) {
		v.AddError(key, "must be a valid string")
		return ""
	}

	return val
}

// ContainsWordChar will ensure that the string has at least a letter or number
func (v *Validator) ContainsWordChar(key, val string, isOptional ...bool) string {
	if len(isOptional) > 0 && isOptional[0] && len(val) == 0 {
		return ""
	}

	if !wordCharRx.MatchString(val) {
		v.AddError(key, "must contain at least a letter or number")
		return ""
	}

	return val
}

// Printable will ensure that all characters in the string can be printed to string (i.e. no control characters)
func (v *Validator) Printable(key, val string, isOptional ...bool) string {
	return v.InverseRegexp(key, val, nonPrintableRx, isOptional...)
}

// PrintableWithNewline will ensure that all characters in the string can be printed to string (i.e. no control characters except for \r\n)
func (v *Validator) PrintableWithNewline(key, val string, isOptional ...bool) string {
	return v.InverseRegexp(key, val, nonPrintableExcludeNewlineRx, isOptional...)
}

// Email will ensure the email address is valid
func (v *Validator) Email(key, email string) string {
	if _, err := mail.ParseAddress(email); err != nil {
		v.AddError(key, "must be a valid email address")
		return ""
	}

	return email
}

// Color will ensure the color is a valid hex color in the form of #000000
func (v *Validator) Color(key, val string, isOptional ...bool) string {
	if len(isOptional) > 0 && isOptional[0] && len(val) == 0 {
		return ""
	}

	if !colorRx.MatchString(val) {
		v.AddError(key, "must be a valid hex color")
		return ""
	}

	return val
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

// GridType will ensure that the string is a valid grid type
func (v *Validator) GridType(key, val string) model.GridType {
	if err := model.IsValidGridType(val); err != nil {
		v.AddError(key, "must be a valid squares type")
		return model.GridType("")
	}

	return model.GridType(val)
}

// Datetime will validate a set of date, time and time zone values and return a time object
func (v *Validator) Datetime(key, dateStr, timeStr, timeZoneOffsetStr string, isOptional ...bool) time.Time {
	if len(isOptional) > 0 && isOptional[0] && ( len(dateStr) == 0 || strings.Index(dateStr, "0000-00-00") == 0 ) {
		return time.Time{}
	}

	if timeStr == "" {
		timeStr = "00:00"
	}

	tz, err := parseTimeZoneOffset(timeZoneOffsetStr)
	if err != nil {
		v.AddError(key, "invalid time zone")
		logrus.WithError(err).WithField("timeZone", timeZoneOffsetStr).Warn("could not parse time zone")
		return time.Time{}
	}

	dt, err := time.Parse("2006-01-02T15:04:05Z0700", fmt.Sprintf("%sT%s:00%s", dateStr, timeStr, tz))
	if err != nil {
		v.AddError(key, "invalid date and/or time")
		return time.Time{}
	}

	return dt
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

// String will returned a concatenated string of error messages (excluding the field name)
func (v *Validator) String() string {
	errors := make([]string, 0)
	for _, err := range v.Errors {
		errors = append(errors, err...)
	}

	return strings.Join(errors, "; ")
}

// parseTimeZoneOffset will take a string time zone offset from JavaScript like 240 or -330 and will convert
// it into a standard ISO 8601 timezone like -0400 or +0530
func parseTimeZoneOffset(s string) (string, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return "", err
	}

	if val == 0 {
		return "+0000", nil
	}

	sig := "+"
	if val > 0 {
		sig = "-"
	}

	val = int(math.Abs(float64(val)))
	h := val / 60
	m := val % 60

	return fmt.Sprintf("%s%02d%02d", sig, h, m), nil
}
