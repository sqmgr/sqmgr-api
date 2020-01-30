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
	dsn           string
	jwtPrivateKey string
	jwtPublicKey  string
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

	viper.SetDefault("dsn", "host=localhost port=5432 user=postgres sslmode=disable")

	if err := viper.ReadInConfig(); err != nil {
		if _, isNotFoundError := err.(viper.ConfigFileNotFoundError); !isNotFoundError {
			return fmt.Errorf("could not read config file: %#v", err)
		}

		logrus.Warn(err)
	}

	instance = &config{
		dsn:           viper.GetString("dsn"),
		jwtPrivateKey: viperGetStringOrFatal("jwt_private_key"),
		jwtPublicKey:  viperGetStringOrFatal("jwt_public_key"),
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
