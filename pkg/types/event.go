// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

// EventType is the kind of event.
type EventType string

var (
	// EventAdd represents the add event.
	EventAdd = EventType("add")
	// EventUpdate represents the update event.
	EventUpdate = EventType("update")
	// EventDelete represents the delete event.
	EventDelete = EventType("delete")
)

// Event describes a specific event generated from the provisioner.
type Event struct {
	Type   EventType
	Object interface{}
	// Tombstone is only valid for delete event,
	// in such a case it stands for the final state
	// of the object.
	Tombstone interface{}
}
