package allocation

import (
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
	JobName   string
	Link      LinkJSON
	TargetURL string
	Label     model.LabelSet
	Collector *collector
}

func (t TargetItem) hash() string {
	return t.JobName + t.TargetURL + t.Label.Fingerprint().String()
}

// Create a struct that holds collector - and jobs for that collector
// This struct will be parsed into endpoint with collector and jobs info

type collector struct {
	Name       string
	NumTargets int
}

// Allocator makes decisions to distribute work among
// a number of OpenTelemetry collectors based on the number of targets.
// Users need to call SetTargets when they have new targets in their
// clusters and call SetCollectors when the collectors have changed.
type Allocator struct {
	// m protects targetsWaiting, collectors, and targetItems for concurrent use.
	m           sync.RWMutex
	collectors  map[string]*collector // all current collectors
	targetItems map[string]*TargetItem

	log logr.Logger
}

// TargetItems returns a shallow copy of the targetItems map.
func (allocator *Allocator) TargetItems() map[string]*TargetItem {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]*TargetItem)
	for k, v := range allocator.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (allocator *Allocator) Collectors() map[string]*collector {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	collectorsCopy := make(map[string]*collector)
	for k, v := range allocator.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// findNextCollector finds the next collector with fewer number of targets.
// This method is called from within SetWaitingTargets and SetCollectors, whose caller
// acquires the needed lock.
func (allocator *Allocator) findNextCollector() *collector {
	var col *collector
	for _, v := range allocator.collectors {
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

// assignTargetToNextCollector assigns a target to the next available collector
func (allocator *Allocator) assignTargetToNextCollector(target *TargetItem) {
	chosenCollector := allocator.findNextCollector()
	targetItem := TargetItem{
		JobName:   target.JobName,
		Link:      LinkJSON{fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(target.JobName))},
		TargetURL: target.TargetURL,
		Label:     target.Label,
		Collector: chosenCollector,
	}
	allocator.targetItems[targetItem.hash()] = &targetItem
	chosenCollector.NumTargets++
	targetsPerCollector.WithLabelValues(chosenCollector.Name).Set(float64(chosenCollector.NumTargets))
}

// allCollectorsPresent checks if all the collectors provided are in the allocator's map
func (allocator *Allocator) allCollectorsPresent(collectors []string) bool {
	if len(collectors) != len(allocator.collectors) {
		return false
	}
	for _, s := range collectors {
		if _, ok := allocator.collectors[s]; !ok {
			return false
		}
	}
	return true
}

// SetWaitingTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (allocator *Allocator) SetWaitingTargets(targets []TargetItem) {
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("SetWaitingTargets"))
	defer timer.ObserveDuration()
	// Dump old data
	allocator.m.Lock()
	defer allocator.m.Unlock()

	// Make the temp map for access
	tempTargetMap := make(map[string]TargetItem, len(targets))
	for _, target := range targets {
		tempTargetMap[target.hash()] = target
	}

	// Check for removals
	for k, target := range allocator.targetItems {
		// if the old target is no longer in the new list, remove it
		if _, ok := tempTargetMap[k]; !ok {
			allocator.collectors[target.Collector.Name].NumTargets--
			delete(allocator.targetItems, k)
		}
	}

	// Check for additions
	for k, target := range tempTargetMap {
		// Do nothing if the item is already there
		if _, ok := allocator.targetItems[k]; ok {
			continue
		} else {
			// Assign a collector to the new target
			allocator.assignTargetToNextCollector(&target)
		}
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (allocator *Allocator) SetCollectors(collectors []string) {
	log := allocator.log.WithValues("component", "opentelemetry-targetallocator")
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("SetCollectors"))
	defer timer.ObserveDuration()

	allocator.m.Lock()
	defer allocator.m.Unlock()
	if len(collectors) == 0 {
		log.Info("No collector instances present")
		return
	} else if allocator.allCollectorsPresent(collectors) {
		log.Info("No changes to the collectors found")
		return
	}

	// Clear existing collectors
	for k := range allocator.collectors {
		delete(allocator.collectors, k)
	}

	// Insert the new collectors
	for _, i := range collectors {
		allocator.collectors[i] = &collector{Name: i, NumTargets: 0}
	}

	// Re-Allocate the existing targets
	for _, item := range allocator.targetItems {
		allocator.assignTargetToNextCollector(item)
	}

	collectorsAllocatable.Set(float64(len(collectors)))
}

func NewAllocator(log logr.Logger) *Allocator {
	return &Allocator{
		log:         log,
		collectors:  make(map[string]*collector),
		targetItems: make(map[string]*TargetItem),
	}
}
