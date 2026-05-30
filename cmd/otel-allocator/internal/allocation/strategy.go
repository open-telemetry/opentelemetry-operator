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

// strategyFactories returns a fresh strategy instance for each allocator.
// Using factories (rather than a singleton map of Strategy values) is
// essential: zone-aware state, per-zone hash rings, and cached collector
// sets all live on the strategy. Sharing one instance across allocators
// would leak that state, and it would also break the new contract that
// "creating an allocator without zone options yields pre-feature
// behavior" — a stale topology from an earlier allocator would silently
// reactivate zone-aware logic on the new one.
var strategyFactories = map[string]func() Strategy{
	leastWeightedStrategyName:     newleastWeightedStrategy,
	consistentHashingStrategyName: newConsistentHashingStrategy,
	perNodeStrategyName:           newPerNodeStrategy,
}

type Option func(Allocator)

type Filter interface {
	Apply([]*target.Item) []*target.Item
}

func WithFilter(filter Filter) Option {
	return func(allocator Allocator) {
		allocator.SetFilter(filter)
	}
}

// WithZoneTopology attaches a ZoneTopology to the allocator. The allocator
// will update it whenever collectors or targets change, so per-zone metrics
// and the /zones API endpoint stay in sync with allocation state.
func WithZoneTopology(zt *ZoneTopology) Option {
	return func(allocator Allocator) {
		allocator.SetZoneTopology(zt)
	}
}

// WithMaxSkew sets the cross-zone spillover threshold for zone-aware
// allocation. 0 disables the check (pure zone affinity).
func WithMaxSkew(maxSkew int) Option {
	return func(allocator Allocator) {
		allocator.SetMaxSkew(maxSkew)
	}
}

func WithFallbackStrategy(fallbackStrategy string) Option {
	factory, ok := strategyFactories[fallbackStrategy]
	if fallbackStrategy != "" && !ok {
		panic(fmt.Errorf("unregistered strategy used as fallback: %s", fallbackStrategy))
	}
	return func(allocator Allocator) {
		if factory == nil {
			allocator.SetFallbackStrategy(nil)
			return
		}
		allocator.SetFallbackStrategy(factory())
	}
}

func New(name string, log logr.Logger, opts ...Option) (Allocator, error) {
	if factory, ok := strategyFactories[name]; ok {
		return newAllocator(log.WithValues("allocator", name), factory(), opts...)
	}
	return nil, fmt.Errorf("unregistered strategy: %s", name)
}

func GetRegisteredAllocatorNames() []string {
	var names []string
	for s := range strategyFactories {
		names = append(names, s)
	}
	return names
}

type Allocator interface {
	SetCollectors(collectors map[string]*Collector)
	SetTargets(targets []*target.Item)
	TargetItems() map[target.ItemHash]*target.Item
	Collectors() map[string]*Collector
	GetTargetsForCollectorAndJob(collector, job string) []*target.Item
	SetFilter(filter Filter)
	SetFallbackStrategy(strategy Strategy)
	// SetZoneTopology attaches (or detaches, with nil) a ZoneTopology
	// tracker. When attached, the allocator mirrors collector and target
	// state into the topology so per-zone metrics and the /zones API
	// endpoint stay consistent. Implementations should be tolerant of
	// being called multiple times.
	SetZoneTopology(zt *ZoneTopology)
	// ZoneTopology returns the currently attached ZoneTopology, or nil
	// when zone awareness is disabled.
	ZoneTopology() *ZoneTopology
	// SetMaxSkew updates the cross-zone spillover threshold. 0 disables
	// the check. Implementations propagate the value to the active
	// strategy via Strategy.SetZoneAwareness.
	SetMaxSkew(maxSkew int)
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
	// SetZoneAwareness enables (or, with nil ZoneTopology, disables) zone-aware
	// allocation. When zt is non-nil, the strategy prefers collectors in the
	// same zone as the target. maxSkew gates cross-zone spillover: when > 0,
	// assigning to a same-zone collector that would push the global skew
	// (max NumTargets - min NumTargets across all collectors) above maxSkew
	// causes the target to be assigned cross-zone instead. Strategies that
	// don't support zone awareness (e.g. per-node) may ignore this call.
	SetZoneAwareness(zt *ZoneTopology, maxSkew int)
}

var _ consistent.Member = Collector{}

// Collector Creates a struct that holds Collector information.
// This struct will be parsed into endpoint with Collector and jobs info.
type Collector struct {
	Name          string
	NodeName      string
	Zone          string
	NumTargets    int
	TargetsPerJob map[string]int
}

func (c Collector) Hash() string {
	return c.Name
}

func (c Collector) String() string {
	return c.Name
}

func NewCollector(name, node, zone string) *Collector {
	return &Collector{Name: name, NodeName: node, Zone: zone, TargetsPerJob: make(map[string]int)}
}
