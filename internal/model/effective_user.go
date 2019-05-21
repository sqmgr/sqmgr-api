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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

var opaqueSalt = os.Getenv("OPAQUE_SALT")

func init() {
	if len(opaqueSalt) == 0 {
		opaqueSalt = "SqMGR-salt"
		logrus.WithField("salt", opaqueSalt).Warn("no OPAQUE_SALT specified, using default")
	}
}

// EffectiveUser provides common user functionality
type EffectiveUser interface {
	JoinGrid(ctx context.Context, s *Pool) error
	IsMemberOf(ctx context.Context, s *Pool) (bool, error)
	IsAdminOf(ctx context.Context, s *Pool) bool
	UserID(ctx context.Context) interface{}
	OpaqueUserID(ctx context.Context) (string, error)
}

func opaqueID(userIdentifier interface{}) (string, error) {
	var id string
	switch val := userIdentifier.(type) {
	case int64:
		id = strconv.FormatInt(val, 10)
	case string:
		id = val
	default:
		panic(fmt.Sprintf("error: invalid type passed to OpaqueID: %T", userIdentifier))
	}

	hasher := sha256.New()
	_, err := hasher.Write([]byte(opaqueSalt + id))
	if err != nil {
		return "", err
	}
	sum := hasher.Sum(nil)

	return hex.EncodeToString(sum[:]), nil
}
