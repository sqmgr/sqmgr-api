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

package config

type config struct {
	url         string
	smtp        string
	fromAddress string
}

var conf = config{
	url:         "http://localhost:8080",
	smtp:        "localhost:25",
	fromAddress: "weters19@gmail.com",
}

func Load() error {
	return nil
}

func GetURL(optionalPath ...string) string {
	if len(optionalPath) > 0 {
		return conf.url + optionalPath[0]
	}

	return conf.url + "/"
}

func GetSMTP() string {
	return conf.smtp
}

func GetFromAddress() string {
	return conf.fromAddress
}
