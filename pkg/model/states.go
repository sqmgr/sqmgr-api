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

// State represents the state of a record
type State string

const (
	// Active means the record is active
	Active State = "active"

	// Pending is specific to a user. They have signed up, but not yet confirmed
	Pending State = "pending"

	// Disabled is when an admin has disabled a record
	Disabled State = "disabled"

	// Deleted is when a user deleted a record
	Deleted State = "deleted"
)
