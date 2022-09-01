package allocation

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/strategy"
)

var (
	collectorsAllocatable = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_collectors_allocatable",
		Help: "Number of collectors the allocator is able to allocate to.",
	})

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

// Allocator makes decisions to distribute work among
// a number of OpenTelemetry collectors based on the number of targets.
// Users need to call SetTargets when they have new targets in their
// clusters and call SetCollectors when the collectors have changed.
type Allocator struct {
	// m protects collectors and targetItems for concurrent use.
	m     sync.RWMutex
	state strategy.State

	log      logr.Logger
	strategy strategy.Allocator
}

// TargetItems returns a shallow copy of the targetItems map.
func (allocator *Allocator) TargetItems() map[string]strategy.TargetItem {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]strategy.TargetItem)
	for k, v := range allocator.state.TargetItems() {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (allocator *Allocator) Collectors() map[string]strategy.Collector {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	collectorsCopy := make(map[string]strategy.Collector)
	for k, v := range allocator.state.Collectors() {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (allocator *Allocator) SetTargets(targets []strategy.TargetItem) {
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("SetTargets"))
	defer timer.ObserveDuration()

	allocator.m.Lock()
	defer allocator.m.Unlock()

	// Make the temp map for access
	tempTargetMap := make(map[string]strategy.TargetItem, len(targets))
	for _, target := range targets {
		tempTargetMap[target.Hash()] = target
	}
	newState := strategy.NewState(allocator.state.Collectors(), tempTargetMap)
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
	newCollectors := map[string]strategy.Collector{}
	for _, s := range collectors {
		newCollectors[s] = strategy.Collector{
			Name:       s,
			NumTargets: 0,
		}
	}
	newState := strategy.NewState(newCollectors, allocator.state.TargetItems())
	allocator.state = allocator.strategy.Allocate(allocator.state, newState)
}

func NewAllocator(log logr.Logger, allocatorStrategy strategy.Allocator) *Allocator {
	return &Allocator{
		log:      log,
		state:    strategy.NewState(make(map[string]strategy.Collector), make(map[string]strategy.TargetItem)),
		strategy: allocatorStrategy,
	}
}
