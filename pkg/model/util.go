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
	"regexp"
	"strings"
)

var hasPortRx = regexp.MustCompile(`:\d+$`)

// Go provides IPs like IP:Port for v4 and [IP]:Port for v6 (i.e., 127.0.0.1:5000 and [::1]:5000)
// Gorilla provides IPs with just the IP for both v4 and v6 (i.e., 127.0.0.1 and ::1)
func ipFromRemoteAddr(remoteAddr string) string {
	if len(remoteAddr) == 0 {
		return ""
	}

	// handle Gorilla ipv6 which does not have port
	if remoteAddr[0] != '[' && strings.Count(remoteAddr, ":") > 1 {
		return remoteAddr
	}

	if !hasPortRx.MatchString(remoteAddr) {
		return remoteAddr
	}

	parts := strings.Split(remoteAddr, ":")
	return strings.Join(parts[0:len(parts)-1], ":")
}
