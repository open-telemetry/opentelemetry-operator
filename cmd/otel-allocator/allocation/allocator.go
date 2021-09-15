package allocation

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/model"
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

	targetsWaiting map[string]TargetItem // temp buffer to keep targets that are waiting to be processed

	collectors map[string]*collector // all current collectors

	TargetItems map[string]*TargetItem

	log logr.Logger
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
		allocator.targetsWaiting[i.JobName+i.TargetURL] = i
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// SetCollectors is called when Collectors are added or removed
func (allocator *Allocator) SetCollectors(collectors []string) {
	log := allocator.log.WithValues("opentelemetry-targetallocator")

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
}

// Reallocate needs to be called to process the new target updates.
// Until Reallocate is called, old targets will be served.
func (allocator *Allocator) AllocateTargets() {
	allocator.m.Lock()
	defer allocator.m.Unlock()
	allocator.removeOutdatedTargets()
	allocator.processWaitingTargets()
}

// ReallocateCollectors reallocates the targets among the new collector instances
func (allocator *Allocator) ReallocateCollectors() {
	allocator.m.Lock()
	defer allocator.m.Unlock()
	allocator.TargetItems = make(map[string]*TargetItem)
	allocator.processWaitingTargets()
}

// removeOutdatedTargets removes targets that are no longer available.
func (allocator *Allocator) removeOutdatedTargets() {
	for k := range allocator.TargetItems {
		if _, ok := allocator.targetsWaiting[k]; !ok {
			allocator.collectors[allocator.TargetItems[k].Collector.Name].NumTargets--
			delete(allocator.TargetItems, k)
		}
	}
}

// processWaitingTargets processes the newly set targets.
func (allocator *Allocator) processWaitingTargets() {
	for k, v := range allocator.targetsWaiting {
		if _, ok := allocator.TargetItems[k]; !ok {
			col := allocator.findNextCollector()
			allocator.TargetItems[k] = &v
			targetItem := TargetItem{
				JobName:   v.JobName,
				Link:      LinkJSON{fmt.Sprintf("/jobs/%s/targets", v.JobName)},
				TargetURL: v.TargetURL,
				Label:     v.Label,
				Collector: col,
			}
			col.NumTargets++
			allocator.TargetItems[v.JobName+v.TargetURL] = &targetItem
		}
	}
}

func NewAllocator(log logr.Logger) *Allocator {
	return &Allocator{
		log:            log,
		targetsWaiting: make(map[string]TargetItem),
		collectors:     make(map[string]*collector),
		TargetItems:    make(map[string]*TargetItem),
	}
}
