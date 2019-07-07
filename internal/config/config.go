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
)

type config struct {
	dsn                string
	fromAddress        string
	jwtPrivateKey      string
	jwtPublicKey       string
	opaqueSalt         string
	recaptchaEnabled   bool
	recaptchaSecretKey string
	recaptchaSiteKey   string
	sessionAuthKey     string
	sessionEncKey      string
	smtp               string
	url                string
}

var instance *config

// URL is the public facing URL of the site
func URL() string {
	mustHaveInstance()
	return instance.url
}

// SMTP returns the location of an smtp server
func SMTP() string {
	mustHaveInstance()
	return instance.smtp
}

// FromAddress will return the email address used in the "from" field in any SqMGR-sent emails
func FromAddress() string {
	mustHaveInstance()
	return instance.fromAddress
}

// SessionAuthKey returns a 64-byte string used for authentication sessions
func SessionAuthKey() string {
	mustHaveInstance()
	return instance.sessionAuthKey
}

// SessionEncKey returns a 32-byte string used for encrypting sessions
func SessionEncKey() string {
	mustHaveInstance()
	return instance.sessionEncKey
}

// DSN returns a database's data source name
func DSN() string {
	mustHaveInstance()
	return instance.dsn
}

// OpaqueSalt returns a salt that is used for obfuscating user IDs
func OpaqueSalt() string {
	mustHaveInstance()
	return instance.opaqueSalt
}

// JWTPublicKey returns the path to a PEM encoded public key
func JWTPublicKey() string {
	mustHaveInstance()
	return instance.jwtPublicKey
}

// JWTPrivateKey returns the path to a PEM encoded private key
func JWTPrivateKey() string {
	mustHaveInstance()
	return instance.jwtPrivateKey
}

// RecaptchaEnabled returns whether recaptcha is currently enabled
func RecaptchaEnabled() bool {
	mustHaveInstance()
	return instance.recaptchaEnabled
}

// RecaptchaSecretKey returns the Google secret key used for reCAPTCHA v3
func RecaptchaSecretKey() string {
	mustHaveInstance()
	return instance.recaptchaSecretKey
}

// RecaptchaSiteKey returns the Google site key used for reCAPTCHA v3
func RecaptchaSiteKey() string {
	mustHaveInstance()
	return instance.recaptchaSiteKey
}

func mustHaveInstance() {
	if instance == nil {
		panic("config: must call Load() first")
	}
}

// Load will setup viper config
func Load() error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/sqmgr")
	viper.SetEnvPrefix("sqmgr_conf")
	viper.BindEnv("dsn")
	viper.BindEnv("recaptcha_site_key")
	viper.BindEnv("recaptcha_secret_key")
	viper.BindEnv("recaptcha_enabled")
	viper.BindEnv("jwt_public_key")
	viper.BindEnv("jwt_private_key")
	viper.BindEnv("opaque_salt")
	viper.BindEnv("session_auth_key")
	viper.BindEnv("session_enc_key")

	viper.SetDefault("dsn", "host=localhost port=5432 user=postgres sslmode=disable")
	viper.SetDefault("recaptcha_enabled", true)
	viper.SetDefault("opaque_salt", "SqMGR-salt")
	viper.SetDefault("url", "http://localhost:8080")
	viper.SetDefault("from_address", "no-reply@sqmgr.com")
	viper.SetDefault("smtp", "localhost:25")

	if err := viper.ReadInConfig(); err != nil {
		if _, isNotFoundError := err.(viper.ConfigFileNotFoundError); !isNotFoundError {
			return fmt.Errorf("could not read config file: %#v", err)
		}

		logrus.Warn(err)
	}

	instance = &config{
		dsn:                viper.GetString("dsn"),
		recaptchaEnabled:   viper.GetBool("recaptcha_enabled"),
		recaptchaSiteKey:   viperGetStringOrWarn("recaptcha_site_key"),
		recaptchaSecretKey: viperGetStringOrWarn("recaptcha_secret_key"),
		jwtPublicKey:       viperGetStringOrFatal("jwt_public_key"),
		jwtPrivateKey:      viperGetStringOrFatal("jwt_private_key"),
		sessionAuthKey:     viperGetStringOrFatal("session_auth_key"),
		sessionEncKey:      viperGetStringOrFatal("session_enc_key"),
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

func viperGetStringOrFatal(key string) string {
	val := viper.GetString(key)
	if val == "" {
		logrus.WithField("key", key).Fatal("configuration key not specified")
	}

	return val
}
