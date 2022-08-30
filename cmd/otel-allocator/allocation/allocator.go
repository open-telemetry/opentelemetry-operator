package allocation

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
)

var (
	collectorsAllocatable = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_collectors_allocatable",
		Help: "Number of collectors the allocator is able to allocate to.",
	})
	targetsPerCollector = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets_per_collector",
		Help: "The number of targets for each collector.",
	}, []string{"collector_name"})
	timeToAssign = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "opentelemetry_allocator_time_to_allocate",
		Help: "The time it takes to allocate",
	}, []string{"method"})
)

/*
	Load balancer will serve on an HTTP server exposing /jobs/<job_id>/targets
	The targets are allocated using the least connection method
	Load balancer will need information about the collectors in order to set the URLs
	Keep a Map of what each collector currently holds and update it based on new scrape target updates
*/

type TargetItem struct {
	JobName       string
	Link          LinkJSON
	TargetURL     string
	Label         model.LabelSet
	CollectorName string
}

func NewTargetItem(jobName string, targetURL string, label model.LabelSet, collectorName string) TargetItem {
	return TargetItem{
		JobName:       jobName,
		Link:          LinkJSON{fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(jobName))},
		TargetURL:     targetURL,
		Label:         label,
		CollectorName: collectorName,
	}
}

func (t TargetItem) hash() string {
	return t.JobName + t.TargetURL + t.Label.Fingerprint().String()
}

// Create a struct that holds collector - and jobs for that collector
// This struct will be parsed into endpoint with collector and jobs info
// This struct can be extended with information like annotations and labels in the future
type collector struct {
	Name       string
	NumTargets int
}

type State struct {
	// collectors is a map from a collector's name to a collector instance
	collectors map[string]collector
	// targetItems is a map from a target item's hash to the target items allocated state
	targetItems map[string]TargetItem
}

func NewState(collectors map[string]collector, targetItems map[string]TargetItem) State {
	return State{collectors: collectors, targetItems: targetItems}
}

type changes[T any] struct {
	additions map[string]T
	removals  map[string]T
}

func diff[T any](current, new map[string]T) changes[T] {
	additions := map[string]T{}
	removals := map[string]T{}
	// Used as a set to check for removed items
	newMembership := map[string]bool{}
	for key, value := range new {
		if _, found := current[key]; !found {
			additions[key] = value
		}
		newMembership[key] = true
	}
	for key, value := range current {
		if _, found := newMembership[key]; !found {
			removals[key] = value
		}
	}
	return changes[T]{
		additions: additions,
		removals:  removals,
	}
}

type AllocatorStrategy interface {
	Allocate(currentState, newState State) State
}

func NewStrategy(kind string) (AllocatorStrategy, error) {
	if kind == "least-weighted" {
		return LeastWeightedStrategy{}, nil
	}
	return nil, errors.New("invalid strategy supplied valid options are [least-weighted]")
}

// Allocator makes decisions to distribute work among
// a number of OpenTelemetry collectors based on the number of targets.
// Users need to call SetTargets when they have new targets in their
// clusters and call SetCollectors when the collectors have changed.
type Allocator struct {
	// m protects collectors and targetItems for concurrent use.
	m     sync.RWMutex
	state State

	log      logr.Logger
	strategy AllocatorStrategy
}

// TargetItems returns a shallow copy of the targetItems map.
func (allocator *Allocator) TargetItems() map[string]TargetItem {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]TargetItem)
	for k, v := range allocator.state.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (allocator *Allocator) Collectors() map[string]collector {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	collectorsCopy := make(map[string]collector)
	for k, v := range allocator.state.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (allocator *Allocator) SetTargets(targets []TargetItem) {
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("SetTargets"))
	defer timer.ObserveDuration()

	allocator.m.Lock()
	defer allocator.m.Unlock()

	// Make the temp map for access
	tempTargetMap := make(map[string]TargetItem, len(targets))
	for _, target := range targets {
		tempTargetMap[target.hash()] = target
	}
	newState := NewState(allocator.state.collectors, tempTargetMap)
	allocator.state = allocator.strategy.Allocate(allocator.state, newState)
}

// SetCollectors sets the set of collectors with key=collectorName, value=CollectorName object.
// This method is called when Collectors are added or removed.
func (allocator *Allocator) SetCollectors(collectors []string) {
	log := allocator.log.WithValues("component", "opentelemetry-targetallocator")
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("SetCollectors"))
	defer timer.ObserveDuration()

	collectorsAllocatable.Set(float64(len(collectors)))
	if len(collectors) == 0 {
		log.Info("No collector instances present")
		return
	}

	allocator.m.Lock()
	defer allocator.m.Unlock()
	newCollectors := map[string]collector{}
	for _, s := range collectors {
		newCollectors[s] = collector{
			Name:       s,
			NumTargets: 0,
		}
	}
	newState := NewState(newCollectors, allocator.state.targetItems)
	allocator.state = allocator.strategy.Allocate(allocator.state, newState)
}

func NewAllocator(log logr.Logger, strategy AllocatorStrategy) *Allocator {
	return &Allocator{
		log: log,
		state: State{
			collectors:  make(map[string]collector),
			targetItems: make(map[string]TargetItem),
		},
		strategy: strategy,
	}
}
