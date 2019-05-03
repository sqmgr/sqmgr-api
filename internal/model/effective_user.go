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

import "context"

// EffectiveUser provides common user functionality
type EffectiveUser interface {
	JoinGrid(ctx context.Context, s *Grid) error
	IsMemberOf(ctx context.Context, s *Grid) (bool, error)
	IsAdminOf(ctx context.Context, s *Grid) bool
	UserID(ctx context.Context) int64
}
