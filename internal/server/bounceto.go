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

package server

import (
	"net/url"
	"strings"
)

// bounceToURL will return a URL that can be used in a bounce-to setting. Because we use handlers.ProxyHeaders(),
// using url.String() will result in something that looks like https:///path?query where what we want is /path?query
func bounceToURL(u *url.URL) string {
	var buf strings.Builder

	path := u.EscapedPath()
	if len(path) == 0 || path[0] != '/' {
		// this shouldn't happen in our context, but just in case
		buf.WriteByte('/')
	}

	buf.WriteString(path)

	if query := u.RawQuery; query != "" {
		buf.WriteByte('?')
		buf.WriteString(query)
	}

	return buf.String()
}
