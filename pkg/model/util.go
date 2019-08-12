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
