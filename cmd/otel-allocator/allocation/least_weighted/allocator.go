package least_weighted

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/strategy"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/utility"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	_ strategy.Allocator = &LeastWeightedAllocator{}
)

/*
	Load balancer will serve on an HTTP server exposing /jobs/<job_id>/targets
	The targets are allocated using the least connection method
	Load balancer will need information about the collectors in order to set the URLs
	Keep a Map of what each collector currently holds and update it based on new scrape target updates
*/

// LeastWeightedAllocator makes decisions to distribute work among
// a number of OpenTelemetry collectors based on the number of targets.
// Users need to call SetTargets when they have new targets in their
// clusters and call SetCollectors when the collectors have changed.
type LeastWeightedAllocator struct {
	// m protects collectors and targetItems for concurrent use.
	m     sync.RWMutex
	state strategy.State

	log logr.Logger
}

// TargetItems returns a shallow copy of the targetItems map.
func (allocator *LeastWeightedAllocator) TargetItems() map[string]*strategy.TargetItem {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]*strategy.TargetItem)
	for k, v := range allocator.state.TargetItems() {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (allocator *LeastWeightedAllocator) Collectors() map[string]*strategy.Collector {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	collectorsCopy := make(map[string]*strategy.Collector)
	for k, v := range allocator.state.Collectors() {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// findNextCollector finds the next collector with fewer number of targets.
// This method is called from within SetTargets and SetCollectors, whose caller
// acquires the needed lock.
func (allocator *LeastWeightedAllocator) findNextCollector() *strategy.Collector {
	var col *strategy.Collector
	for _, v := range allocator.state.Collectors() {
		// If the initial collector is empty, set the initial collector to the first element of map
		if col == nil {
			col = v
		} else {
			if v.NumTargets < col.NumTargets {
				col = v
			}
		}
	}
	return col
}

// addTargetToTargetItems assigns a target to the next available collector and adds it to the allocator's targetItems
// This method is called from within SetTargets and SetCollectors, whose caller acquires the needed lock.
// This is only called after the collectors are cleared or when a new target has been found in the tempTargetMap
func (allocator *LeastWeightedAllocator) addTargetToTargetItems(target *strategy.TargetItem) {
	chosenCollector := allocator.findNextCollector()
	targetItem := &strategy.TargetItem{
		JobName:       target.JobName,
		Link:          strategy.LinkJSON{Link: fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(target.JobName))},
		TargetURL:     target.TargetURL,
		Label:         target.Label,
		CollectorName: chosenCollector.Name,
	}
	allocator.state.SetTargetItem(targetItem.Hash(), targetItem)
	chosenCollector.NumTargets++
	strategy.TargetsPerCollector.WithLabelValues(chosenCollector.Name).Set(float64(chosenCollector.NumTargets))
}

func (allocator *LeastWeightedAllocator) handleTargets(diff utility.Changes[*strategy.TargetItem]) {
	// Check for removals
	for k, target := range allocator.state.TargetItems() {
		// if the current target is in the removals list
		if _, ok := diff.Removals()[k]; ok {
			c := allocator.state.Collectors()[target.CollectorName]
			c.NumTargets--
			allocator.state.RemoveTargetItem(k)
			strategy.TargetsPerCollector.WithLabelValues(target.CollectorName).Set(float64(c.NumTargets))
		}
	}

	// Check for additions
	for k, target := range diff.Additions() {
		// Do nothing if the item is already there
		if _, ok := allocator.state.TargetItems()[k]; ok {
			continue
		} else {
			// Assign new set of collectors with the one different name
			allocator.addTargetToTargetItems(target)
		}
	}
}

func (allocator *LeastWeightedAllocator) handleCollectors(diff utility.Changes[*strategy.Collector]) {
	// Clear existing collectors
	for _, k := range diff.Removals() {
		allocator.state.RemoveCollector(k.Name)
		strategy.TargetsPerCollector.WithLabelValues(k.Name).Set(0)
	}
	// Insert the new collectors
	for _, i := range diff.Additions() {
		allocator.state.SetCollector(i.Name, &strategy.Collector{Name: i.Name, NumTargets: 0})
	}

	// find targets which need to be redistributed
	var redistribute []*strategy.TargetItem
	for _, item := range allocator.state.TargetItems() {
		for _, s := range diff.Removals() {
			if item.CollectorName == s.Name {
				redistribute = append(redistribute, item)
			}
		}
	}
	// Re-Allocate the existing targets
	for _, item := range redistribute {
		allocator.addTargetToTargetItems(item)
	}
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (allocator *LeastWeightedAllocator) SetTargets(targets map[string]*strategy.TargetItem) {
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("SetTargets"))
	defer timer.ObserveDuration()

	allocator.m.Lock()
	defer allocator.m.Unlock()

	// Check for target changes
	targetsDiff := utility.DiffMaps(allocator.state.TargetItems(), targets)
	// If there are any additions or removals
	if len(targetsDiff.Additions()) != 0 || len(targetsDiff.Removals()) != 0 {
		allocator.handleTargets(targetsDiff)
	}
	return
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (allocator *LeastWeightedAllocator) SetCollectors(collectors map[string]*strategy.Collector) {
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

	// Check for collector changes
	collectorsDiff := utility.DiffMaps(allocator.state.Collectors(), collectors)
	if len(collectorsDiff.Additions()) != 0 || len(collectorsDiff.Removals()) != 0 {
		allocator.handleCollectors(collectorsDiff)
	}
	return
}

func NewAllocator(log logr.Logger) strategy.Allocator {
	return &LeastWeightedAllocator{
		log:   log,
		state: strategy.NewState(make(map[string]*strategy.Collector), make(map[string]*strategy.TargetItem)),
	}
}

func init() {
	err := strategy.Register("least-weighted", NewAllocator)
	if err != nil {
		os.Exit(1)
	}
}
