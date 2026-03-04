// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

var _ allocation.Allocator = &mockAllocator{}

// mockAllocator implements the Allocator interface, but all funcs other than
// TargetItems() are a no-op.
type mockAllocator struct {
	targetItems map[target.ItemHash]*target.Item
}

func (*mockAllocator) SetCollectors(map[string]*allocation.Collector)             {}
func (*mockAllocator) SetTargets([]*target.Item)                                  {}
func (*mockAllocator) Collectors() map[string]*allocation.Collector               { return nil }
func (*mockAllocator) GetTargetsForCollectorAndJob(string, string) []*target.Item { return nil }
func (*mockAllocator) SetFilter(allocation.Filter)                                {}
func (*mockAllocator) SetFallbackStrategy(allocation.Strategy)                    {}

func (m *mockAllocator) TargetItems() map[target.ItemHash]*target.Item {
	return m.targetItems
}
