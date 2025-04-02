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

func (m *mockAllocator) SetCollectors(_ map[string]*allocation.Collector)               {}
func (m *mockAllocator) SetTargets(_ []*target.Item)                                    {}
func (m *mockAllocator) Collectors() map[string]*allocation.Collector                   { return nil }
func (m *mockAllocator) GetTargetsForCollectorAndJob(_ string, _ string) []*target.Item { return nil }
func (m *mockAllocator) SetFilter(_ allocation.Filter)                                  {}
func (m *mockAllocator) SetFallbackStrategy(_ allocation.Strategy)                      {}

func (m *mockAllocator) TargetItems() map[target.ItemHash]*target.Item {
	return m.targetItems
}
