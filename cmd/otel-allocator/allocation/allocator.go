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
	Load balancer will serve on an HTTP server exposing /jobs/<job_id>/targets <- these are configured using least connection
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
// clusters and call Reshard to process the new targets and reshard.
type Allocator struct {
	m sync.RWMutex

	targetsWaiting map[string]TargetItem // temp buffer to keep targets that are waiting to be processed

	collectors map[string]*collector // all current collectors

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

// findNextCollector finds the next collector with less number of targets.
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

// SetTargets accepts the a list of targets that will be used to make
// load balancing decisions. This method should be called when where are
// new targets discovered or existing targets are shutdown.
func (allocator *Allocator) SetWaitingTargets(targets []TargetItem) {
	// Dump old data
	allocator.m.Lock()
	defer allocator.m.Unlock()
	allocator.targetsWaiting = make(map[string]TargetItem, len(targets))
	// Set new data
	for _, i := range targets {
		allocator.targetsWaiting[i.hash()] = i
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// SetCollectors is called when Collectors are added or removed
func (allocator *Allocator) SetCollectors(collectors []string) {
	log := allocator.log.WithValues("component", "opentelemetry-targetallocator")

	allocator.m.Lock()
	defer allocator.m.Unlock()
	if len(collectors) == 0 {
		log.Info("No collector instances present")
		return
	}
	for k := range allocator.collectors {
		delete(allocator.collectors, k)
	}

	for _, i := range collectors {
		allocator.collectors[i] = &collector{Name: i, NumTargets: 0}
	}
	collectorsAllocatable.Set(float64(len(collectors)))
}

// Reallocate needs to be called to process the new target updates.
// Until Reallocate is called, old targets will be served.
func (allocator *Allocator) AllocateTargets() {
	allocator.m.Lock()
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("AllocateTargets"))
	defer timer.ObserveDuration()
	defer allocator.m.Unlock()
	allocator.removeOutdatedTargets()
	allocator.processWaitingTargets()
}

// ReallocateCollectors reallocates the targets among the new collector instances
func (allocator *Allocator) ReallocateCollectors() {
	allocator.m.Lock()
	timer := prometheus.NewTimer(timeToAssign.WithLabelValues("ReallocateCollectors"))
	defer timer.ObserveDuration()
	defer allocator.m.Unlock()
	allocator.targetItems = make(map[string]*TargetItem)
	allocator.processWaitingTargets()
}

// removeOutdatedTargets removes targets that are no longer available.
func (allocator *Allocator) removeOutdatedTargets() {
	for k := range allocator.targetItems {
		if _, ok := allocator.targetsWaiting[k]; !ok {
			allocator.collectors[allocator.targetItems[k].Collector.Name].NumTargets--
			delete(allocator.targetItems, k)
		}
	}
}

// processWaitingTargets processes the newly set targets.
func (allocator *Allocator) processWaitingTargets() {
	for k, v := range allocator.targetsWaiting {
		if _, ok := allocator.targetItems[k]; !ok {
			col := allocator.findNextCollector()
			allocator.targetItems[k] = &v
			targetItem := TargetItem{
				JobName:   v.JobName,
				Link:      LinkJSON{fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(v.JobName))},
				TargetURL: v.TargetURL,
				Label:     v.Label,
				Collector: col,
			}
			col.NumTargets++
			targetsPerCollector.WithLabelValues(col.Name).Set(float64(col.NumTargets))
			allocator.targetItems[v.hash()] = &targetItem
		}
	}
}

func NewAllocator(log logr.Logger) *Allocator {
	return &Allocator{
		log:            log,
		targetsWaiting: make(map[string]TargetItem),
		collectors:     make(map[string]*collector),
		targetItems:    make(map[string]*TargetItem),
	}
}
