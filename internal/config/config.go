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

package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type config struct {
	dsn                string
	jwtPrivateKey      string
	jwtPublicKey       string
	auth0JWKSURL       string
	auth0MgmtDomain    string
	auth0MgmtClientID  string
	auth0MgmtClientSec string
	corsAllowedOrigins []string
}

var instance *config

// DSN returns a database's data source name
func DSN() string {
	mustHaveInstance()
	return instance.dsn
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

// Auth0JWKSURL returns the URL to the Auth0 JWKS endpoint
func Auth0JWKSURL() string {
	mustHaveInstance()
	return instance.auth0JWKSURL
}

// Auth0MgmtDomain returns the Auth0 Management API domain
func Auth0MgmtDomain() string {
	mustHaveInstance()
	return instance.auth0MgmtDomain
}

// Auth0MgmtClientID returns the Auth0 Management API client ID
func Auth0MgmtClientID() string {
	mustHaveInstance()
	return instance.auth0MgmtClientID
}

// Auth0MgmtClientSecret returns the Auth0 Management API client secret
func Auth0MgmtClientSecret() string {
	mustHaveInstance()
	return instance.auth0MgmtClientSec
}

// CORSAllowedOrigins returns the list of allowed CORS origins
func CORSAllowedOrigins() []string {
	mustHaveInstance()
	return instance.corsAllowedOrigins
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
	_ = viper.BindEnv("dsn")
	_ = viper.BindEnv("jwt_private_key")
	_ = viper.BindEnv("jwt_public_key")
	_ = viper.BindEnv("auth0_jwks_url")
	_ = viper.BindEnv("auth0_mgmt_domain")
	_ = viper.BindEnv("auth0_mgmt_client_id")
	_ = viper.BindEnv("auth0_mgmt_client_secret")
	_ = viper.BindEnv("cors_allowed_origins")

	viper.SetDefault("dsn", "host=localhost port=5432 user=postgres sslmode=disable")
	viper.SetDefault("auth0_jwks_url", "https://sqmgr.auth0.com/.well-known/jwks.json")
	viper.SetDefault("cors_allowed_origins", "https://sqmgr.com,https://www.sqmgr.com,http://localhost:8080")

	if err := viper.ReadInConfig(); err != nil {
		if _, isNotFoundError := err.(viper.ConfigFileNotFoundError); !isNotFoundError {
			return fmt.Errorf("could not read config file: %#v", err)
		}

		logrus.Warn(err)
	}

	var corsOrigins []string
	for _, origin := range strings.Split(viper.GetString("cors_allowed_origins"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			corsOrigins = append(corsOrigins, origin)
		}
	}

	instance = &config{
		dsn:                viper.GetString("dsn"),
		jwtPrivateKey:      viperGetStringOrFatal("jwt_private_key"),
		jwtPublicKey:       viperGetStringOrFatal("jwt_public_key"),
		auth0JWKSURL:       viper.GetString("auth0_jwks_url"),
		auth0MgmtDomain:    viper.GetString("auth0_mgmt_domain"),
		auth0MgmtClientID:  viper.GetString("auth0_mgmt_client_id"),
		auth0MgmtClientSec: viper.GetString("auth0_mgmt_client_secret"),
		corsAllowedOrigins: corsOrigins,
	}

	return nil
}

func viperGetStringOrFatal(key string) string {
	val := viper.GetString(key)
	if val == "" {
		logrus.WithField("key", key).Fatal("configuration key not specified")
	}

	return val
}
