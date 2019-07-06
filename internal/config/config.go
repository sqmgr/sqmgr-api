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

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

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

// Load will load the config
func Load() error {
	return nil
}

// GetURL will get the URL to the site
func GetURL(optionalPath ...string) string {
	url := os.Getenv("URL")
	if len(url) == 0 {
		url = conf.url
	}

	if len(optionalPath) > 0 {
		return url + optionalPath[0]
	}

	return url + "/"
}

// GetSMTP will load the SMTP configuration
func GetSMTP() string {
	if smtp := os.Getenv("SMTP"); len(smtp) > 0 {
		return smtp
	}

	return conf.smtp
}

// GetFromAddress will get the address emails should appear from
func GetFromAddress() string {
	if from := os.Getenv("FROM_ADDRESS"); len(from) > 0 {
		return from
	}

	return conf.fromAddress
}

var instance *Config

type Config struct {
	recaptchaEnabled bool
	recaptchaSiteKey string
	recaptchaSecretKey string
}

func RecaptchaEnabled() bool {
	if instance == nil {
		panic("Setup() not called")
	}

	return instance.recaptchaEnabled
}

func RecaptchaSecretKey() string {
	if instance == nil {
		panic("Setup() not called")
	}

	return instance.recaptchaSecretKey
}

func RecaptchaSiteKey() string {
	if instance == nil {
		panic("Setup() not called")
	}

	return instance.recaptchaSiteKey
}


// Setup will setup viper config
func Setup() error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/sqmgr")
	viper.SetEnvPrefix("sqmgr_conf")
	viper.BindEnv("recaptcha_site_key")
	viper.BindEnv("recaptcha_secret_key")
	viper.BindEnv("recaptcha_enabled")

	viper.SetDefault("recaptcha_enabled", true)

	if err := viper.ReadInConfig(); err != nil {
		if _, isNotFoundError := err.(viper.ConfigFileNotFoundError); !isNotFoundError {
			return fmt.Errorf("could not read config file: %#v", err)
		}

		logrus.Warn(err)
	}

	instance = &Config{
		recaptchaEnabled: viper.GetBool("recaptcha_enabled"),
		recaptchaSiteKey: viperGetStringOrWarn("recaptcha_site_key"),
		recaptchaSecretKey: viperGetStringOrWarn("recaptcha_secret_key"),
	}

	return nil
}

func viperGetStringOrWarn(key string) string {
	val := viper.GetString(key)
	if val == "" {
		logrus.WithField("key", key).Warn("configuration key not specified")
	}

	return val
}
