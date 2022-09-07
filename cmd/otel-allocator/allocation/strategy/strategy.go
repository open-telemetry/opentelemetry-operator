package strategy

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type AllocatorProvider func(log logr.Logger) Allocator

var (
	registry = map[string]AllocatorProvider{}

	// TargetsPerCollector records how many targets have been assigned to each collector
	// It is currently the responsibility of the strategy to track this information.
	TargetsPerCollector = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets_per_collector",
		Help: "The number of targets for each collector.",
	}, []string{"collector_name"})
)

func New(name string, log logr.Logger) (Allocator, error) {
	if p, ok := registry[name]; ok {
		return p(log), nil
	}
	return nil, errors.New(fmt.Sprintf("unregistered strategy: %s", name))
}

func Register(name string, provider AllocatorProvider) error {
	if _, ok := registry[name]; ok {
		return errors.New("already registered")
	}
	registry[name] = provider
	return nil
}

type Allocator interface {
	SetCollectors(collectors map[string]*Collector)
	SetTargets(targets map[string]*TargetItem)
	TargetItems() map[string]*TargetItem
	Collectors() map[string]*Collector
}
