// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
