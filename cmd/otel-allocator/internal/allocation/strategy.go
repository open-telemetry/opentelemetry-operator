// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"

	"github.com/buraksezer/consistent"
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// strategyBuilder constructs a Strategy from the allocation strategy configuration. A builder reads only
// the configuration relevant to its strategy and constructs any strategies its strategy depends on
// (e.g. a fallback strategy), which are then injected into the strategy's constructor.
type strategyBuilder func(config StrategyConfig) (Strategy, error)

// strategyBuilders returns the registry of strategy builders keyed by strategy name. It is a function
// rather than a package-level variable so builders can call buildStrategy for the strategies they depend
// on without creating an initialization cycle.
func strategyBuilders() map[string]strategyBuilder {
	return map[string]strategyBuilder{
		leastWeightedStrategyName:     func(StrategyConfig) (Strategy, error) { return newleastWeightedStrategy(), nil },
		consistentHashingStrategyName: func(StrategyConfig) (Strategy, error) { return newConsistentHashingStrategy(), nil },
		perNodeStrategyName:           buildPerNodeStrategy,
	}
}

// buildStrategy constructs the named strategy, resolving and injecting any strategies it depends on.
func buildStrategy(name string, config StrategyConfig) (Strategy, error) {
	build, ok := strategyBuilders()[name]
	if !ok {
		return nil, fmt.Errorf("unregistered strategy: %s", name)
	}
	return build(config)
}

// buildFallbackStrategy constructs the strategy configured as a fallback. FallbackStrategyConfig has no
// sections carrying fallbacks of their own, so a fallback strategy can never have a fallback itself,
// keeping fallback chains bounded to a single level by construction.
func buildFallbackStrategy(config FallbackStrategyConfig) (Strategy, error) {
	return buildStrategy(config.Name, StrategyConfig{
		ConsistentHashing: config.ConsistentHashing,
		LeastWeighted:     config.LeastWeighted,
	})
}

// Option configures the allocator constructed by New.
type Option func(*allocatorOptions)

type allocatorOptions struct {
	strategyConfig StrategyConfig
}

// StrategyConfig holds the configuration for the allocation strategies. Each strategy has its own
// section because strategies accept different configuration options.
type StrategyConfig struct {
	ConsistentHashing ConsistentHashingStrategyConfig
	LeastWeighted     LeastWeightedStrategyConfig
	PerNode           PerNodeStrategyConfig
}

// ConsistentHashingStrategyConfig holds the configuration options for the consistent-hashing strategy.
type ConsistentHashingStrategyConfig struct{}

// LeastWeightedStrategyConfig holds the configuration options for the least-weighted strategy.
type LeastWeightedStrategyConfig struct{}

// PerNodeStrategyConfig holds the configuration options for the per-node strategy.
type PerNodeStrategyConfig struct {
	// FallbackStrategy configures the strategy used for targets the per-node strategy can't assign on
	// its own, for example targets which don't have a node label. If nil, such targets are left unassigned.
	FallbackStrategy *FallbackStrategyConfig
}

// FallbackStrategyConfig holds the name and options of a strategy used as a fallback. It mirrors
// StrategyConfig, except that strategies used as fallbacks can't have fallbacks of their own, which
// keeps fallback chains bounded to a single level.
type FallbackStrategyConfig struct {
	Name              string
	ConsistentHashing ConsistentHashingStrategyConfig
	LeastWeighted     LeastWeightedStrategyConfig
}

// WithStrategyConfig sets the configuration used to construct the allocator's strategy.
func WithStrategyConfig(config StrategyConfig) Option {
	return func(o *allocatorOptions) {
		o.strategyConfig = config
	}
}

func New(name string, log logr.Logger, opts ...Option) (Allocator, error) {
	var options allocatorOptions
	for _, opt := range opts {
		opt(&options)
	}
	strategy, err := buildStrategy(name, options.strategyConfig)
	if err != nil {
		return nil, err
	}
	return newAllocator(log.WithValues("allocator", name), strategy)
}

func GetRegisteredAllocatorNames() []string {
	var names []string
	for s := range strategyBuilders() {
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
}

type Strategy interface {
	GetCollectorForTarget(map[string]*Collector, *target.Item) (*Collector, error)
	// SetCollectors exists for strategies where changing the collector set is potentially an expensive operation.
	// The caller must guarantee that the collectors map passed in GetCollectorForTarget is consistent with the latest
	// SetCollectors call. Strategies which don't need this information can just ignore it.
	SetCollectors(map[string]*Collector)
	GetName() string
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
