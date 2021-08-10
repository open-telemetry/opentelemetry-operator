package allocation

import (
	"log"
	"sync"
)

/*
	Load balancer will serve on an HTTP server exposing /jobs/<job_id>/targets <- these are configured using least connection
	Load balancer will need information about the collectors in order to set the URLs
	Keep a Map of what each collector currently holds and update it based on new scrape target updates
*/

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

	targetItems map[string]*TargetItem
}

// findNextCollector finds the next collector with less number of targets.
func (allocator *Allocator) findNextCollector() *collector {

	col := &collector{}
	for _, v := range allocator.collectors {
		// If the initial collector is empty, set the initial collector to the first element of map
		if col.Name == "" {
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
	allocator.targetsWaiting = make(map[string]TargetItem)
	// Set new data
	for _, i := range targets {
		allocator.targetsWaiting[i.JobName+i.TargetURL] = i
	}
	allocator.m.Unlock()
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// SetCollectors is called when Collectors are added or removed
func (allocator *Allocator) SetCollectors(collectors []string) {
	if len(collectors) == 0 {
		log.Println("no collector instances present")
		return
	}
	allocator.m.Lock()
	for k := range allocator.collectors {
		delete(allocator.collectors, k)
	}

	for _, i := range collectors {
		allocator.collectors[i] = &collector{Name: i, NumTargets: 0}
	}

	allocator.m.Unlock()
}

// Reallocate needs to be called to process the new target updates.
// Until Reallocate is called, old targets will be served.
func (allocator *Allocator) Reallocate() {
	allocator.m.Lock()
	allocator.removeOutdatedTargets()
	allocator.processWaitingTargets()
	allocator.m.Unlock()
}

// ReallocateCollectors reallocates the targets among the new collector instances
func (allocator *Allocator) ReallocateCollectors() {
	allocator.m.Lock()
	allocator.targetItems = make(map[string]*TargetItem)
	allocator.processWaitingTargets()
	allocator.m.Unlock()
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
				Link:      linkJSON{"/jobs/" + v.JobName + "/targets"},
				TargetURL: v.TargetURL,
				Label:     v.Label,
				Collector: col,
			}
			col.NumTargets++
			allocator.targetItems[v.JobName+v.TargetURL] = &targetItem
		}
	}
}

func NewAllocator() *Allocator {
	return &Allocator{
		targetsWaiting: make(map[string]TargetItem),
		collectors:     make(map[string]*collector),
		targetItems:    make(map[string]*TargetItem),
	}
}
