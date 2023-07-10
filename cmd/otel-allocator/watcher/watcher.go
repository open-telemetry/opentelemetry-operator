// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package watcher

import (
	"context"

	promconfig "github.com/prometheus/prometheus/config"
)

type Watcher interface {
	// Watch watcher and supply channels which will receive change events
	Watch(upstreamEvents chan Event, upstreamErrors chan error) error
	LoadConfig(ctx context.Context) (*promconfig.Config, error)
	Close() error
}

type Event struct {
	Source  EventSource
	Watcher Watcher
}

type EventSource int

const (
	EventSourceConfigMap EventSource = iota
	EventSourcePrometheusCR
)

var (
	eventSourceToString = map[EventSource]string{
		EventSourceConfigMap:    "EventSourceConfigMap",
		EventSourcePrometheusCR: "EventSourcePrometheusCR",
	}
)

func (e EventSource) String() string {
	return eventSourceToString[e]
}
