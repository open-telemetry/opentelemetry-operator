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
	m sync.Mutex

	//targetsWaiting map[string]TargetItem // temp buffer to keep targets that are waiting to be processed

	collectors map[string]*collector // all current collectors

	targetItems map[string]*TargetItem

	log logr.Logger
}

func (allocator *Allocator) TargetItems() map[string]*TargetItem {
	return allocator.targetItems
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

// allCollectorsPresent checks if all of the collectors provided are in the allocator's map
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

// SetTargets accepts the a list of targets that will be used to make
// load balancing decisions. This method should be called when where are
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
			col := allocator.findNextCollector()
			targetItem := TargetItem{
				JobName:   target.JobName,
				Link:      LinkJSON{fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(target.JobName))},
				TargetURL: target.TargetURL,
				Label:     target.Label,
				Collector: col,
			}
			col.NumTargets++
			targetsPerCollector.WithLabelValues(col.Name).Set(float64(col.NumTargets))
			allocator.targetItems[k] = &targetItem
		}
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// SetCollectors is called when Collectors are added or removed
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
	for k := range allocator.collectors {
		delete(allocator.collectors, k)
	}

	for _, i := range collectors {
		allocator.collectors[i] = &collector{Name: i, NumTargets: 0}
	}
	for k, _ := range allocator.targetItems {
		chosenCollector := allocator.findNextCollector()
		allocator.targetItems[k].Collector = chosenCollector
		chosenCollector.NumTargets++
		targetsPerCollector.WithLabelValues(chosenCollector.Name).Set(float64(chosenCollector.NumTargets))
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
