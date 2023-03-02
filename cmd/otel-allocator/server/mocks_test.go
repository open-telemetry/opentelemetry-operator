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

package server

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var _ allocation.Allocator = &mockAllocator{}

// mockAllocator implements the Allocator interface, but all funcs other than
// TargetItems() are a no-op.
type mockAllocator struct {
	targetItems map[string]*target.Item
}

func (m *mockAllocator) SetCollectors(_ map[string]*allocation.Collector)               {}
func (m *mockAllocator) SetTargets(_ map[string]*target.Item)                           {}
func (m *mockAllocator) Collectors() map[string]*allocation.Collector                   { return nil }
func (m *mockAllocator) GetTargetsForCollectorAndJob(_ string, _ string) []*target.Item { return nil }
func (m *mockAllocator) SetFilter(_ allocation.Filter)                                  {}

func (m *mockAllocator) TargetItems() map[string]*target.Item {
	return m.targetItems
}
