// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"

	"github.com/buraksezer/consistent"
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

type AllocatorProvider func(log logr.Logger, opts ...Option) Allocator

var (
	strategies = map[string]Strategy{
		leastWeightedStrategyName:     newleastWeightedStrategy(),
		consistentHashingStrategyName: newConsistentHashingStrategy(),
		perNodeStrategyName:           newPerNodeStrategy(),
	}
)

type Option func(Allocator)

type Filter interface {
	Apply([]*target.Item) []*target.Item
}

func WithFilter(filter Filter) Option {
	return func(allocator Allocator) {
		allocator.SetFilter(filter)
	}
}

func WithFallbackStrategy(fallbackStrategy string) Option {
	var strategy, ok = strategies[fallbackStrategy]
	if fallbackStrategy != "" && !ok {
		panic(fmt.Errorf("unregistered strategy used as fallback: %s", fallbackStrategy))
	}
	return func(allocator Allocator) {
		allocator.SetFallbackStrategy(strategy)
	}
}

func New(name string, log logr.Logger, opts ...Option) (Allocator, error) {
	if strategy, ok := strategies[name]; ok {
		return newAllocator(log.WithValues("allocator", name), strategy, opts...)
	}
	return nil, fmt.Errorf("unregistered strategy: %s", name)
}

func GetRegisteredAllocatorNames() []string {
	var names []string
	for s := range strategies {
		names = append(names, s)
	}
	return names
}

type Allocator interface {
	SetCollectors(collectors map[string]*Collector)
	SetTargets(targets []*target.Item)
	TargetItems() map[target.ItemHash]*target.Item
	Collectors() map[string]*Collector
	GetTargetsForCollectorAndJob(collector string, job string) []*target.Item
	SetFilter(filter Filter)
	SetFallbackStrategy(strategy Strategy)
}

type Strategy interface {
	GetCollectorForTarget(map[string]*Collector, *target.Item) (*Collector, error)
	// SetCollectors exists for strategies where changing the collector set is potentially an expensive operation.
	// The caller must guarantee that the collectors map passed in GetCollectorForTarget is consistent with the latest
	// SetCollectors call. Strategies which don't need this information can just ignore it.
	SetCollectors(map[string]*Collector)
	GetName() string
	// SetFallbackStrategy adds fallback strategy for strategies whose main allocation method can sometimes leave targets unassigned
	SetFallbackStrategy(Strategy)
}

var _ consistent.Member = Collector{}

// Collector Creates a struct that holds Collector information.
// This struct will be parsed into endpoint with Collector and jobs info.
// This struct can be extended with information like annotations and labels in the future.
type Collector struct {
	Name          string
	NodeName      string
	NumTargets    int
	TargetsPerJob map[string]int
}

func (c Collector) Hash() string {
	return c.Name
}

func (c Collector) String() string {
	return c.Name
}

func NewCollector(name, node string) *Collector {
	return &Collector{Name: name, NodeName: node, TargetsPerJob: make(map[string]int)}
}
