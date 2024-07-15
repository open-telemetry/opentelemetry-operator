// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allocation

import (
	"sync"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/diff"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
)

var _ Allocator = &jobAverageAllocator{}

const jobAverageStrategyName = "job-average"

// jobAverageAllocator makes decisions to distribute job' targets
// among a number of OpenTelemetry collectors based on the number
// of targets that have been allocated to this job and the total
// number of all targets.
// Users need to call SetTargets when they have new targets in their
// clusters and call SetCollectors when the collectors have changed.
type jobAverageAllocator struct {
	// m protects collectors and targetItems for concurrent use.
	m sync.RWMutex
	// collectors is a map from a Collector's name to a Collector instance
	collectors map[string]*Collector
	// targetItems is a map from a target item's hash to the target items allocated state
	targetItems map[string]*target.Item

	// collectorKey -> job -> target item hash -> true
	targetItemsPerJobPerCollector map[string]map[string]map[string]bool

	log logr.Logger

	filter Filter
}

// SetFilter sets the filtering hook to use.
func (allocator *jobAverageAllocator) SetFilter(filter Filter) {
	allocator.filter = filter
}

func (allocator *jobAverageAllocator) GetTargetsForCollectorAndJob(collector string, job string) []*target.Item {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	if _, ok := allocator.targetItemsPerJobPerCollector[collector]; !ok {
		return []*target.Item{}
	}
	if _, ok := allocator.targetItemsPerJobPerCollector[collector][job]; !ok {
		return []*target.Item{}
	}
	targetItemsCopy := make([]*target.Item, len(allocator.targetItemsPerJobPerCollector[collector][job]))
	index := 0
	for targetHash := range allocator.targetItemsPerJobPerCollector[collector][job] {
		targetItemsCopy[index] = allocator.targetItems[targetHash]
		index++
	}
	return targetItemsCopy
}

// TargetItems returns a shallow copy of the targetItems map.
func (allocator *jobAverageAllocator) TargetItems() map[string]*target.Item {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]*target.Item)
	for k, v := range allocator.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (allocator *jobAverageAllocator) Collectors() map[string]*Collector {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	collectorsCopy := make(map[string]*Collector)
	for k, v := range allocator.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// findNextCollector finds the next collector with fewer number of job targets, if number of job targets
// is equal, than finds the next collector with fewer number of total targets
// This method is called from within SetTargets and SetCollectors, whose caller
// acquires the needed lock. This method assumes there are is at least 1 collector set.
// INVARIANT: allocator.collectors must have at least 1 collector set.
func (allocator *jobAverageAllocator) findNextCollector(tg *target.Item) *Collector {
	var chosen *Collector
	leastJobTargetNum := 0
	for _, cur := range allocator.collectors {
		curJobTargetNum := 0
		if allocator.targetItemsPerJobPerCollector[cur.Name] != nil && allocator.targetItemsPerJobPerCollector[cur.Name][tg.JobName] != nil {
			curJobTargetNum = len(allocator.targetItemsPerJobPerCollector[cur.Name][tg.JobName])
		}
		// If the initial collector is empty, set the initial collector to the first element of map
		if chosen == nil {
			chosen = cur
			leastJobTargetNum = curJobTargetNum
		} else if curJobTargetNum < leastJobTargetNum || (curJobTargetNum == leastJobTargetNum && cur.NumTargets < chosen.NumTargets) {
			chosen = cur
			leastJobTargetNum = curJobTargetNum
		}
	}
	return chosen
}

// addCollectorTargetItemMapping keeps track of which collector has which jobs and targets
// this allows the allocator to respond without any extra allocations to http calls. The caller of this method
// has to acquire a lock.
func (allocator *jobAverageAllocator) addCollectorTargetItemMapping(tg *target.Item) {
	if allocator.targetItemsPerJobPerCollector[tg.CollectorName] == nil {
		allocator.targetItemsPerJobPerCollector[tg.CollectorName] = make(map[string]map[string]bool)
	}
	if allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] == nil {
		allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] = make(map[string]bool)
	}
	allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName][tg.Hash()] = true
}

// addTargetToTargetItems assigns a target to the next available collector and adds it to the allocator's targetItems
// This method is called from within SetTargets and SetCollectors, which acquire the needed lock.
// This is only called after the collectors are cleared or when a new target has been found in the tempTargetMap.
// INVARIANT: allocator.collectors must have at least 1 collector set.
// NOTE: by not creating a new target item, there is the potential for a race condition where we modify this target
// item while it's being encoded by the server JSON handler.
func (allocator *jobAverageAllocator) addTargetToTargetItems(tg *target.Item) {
	chosenCollector := allocator.findNextCollector(tg)
	tg.CollectorName = chosenCollector.Name
	allocator.targetItems[tg.Hash()] = tg
	allocator.addCollectorTargetItemMapping(tg)
	chosenCollector.NumTargets++
	TargetsPerCollector.WithLabelValues(chosenCollector.Name, jobAverageStrategyName).Set(float64(chosenCollector.NumTargets))

}

// handleTargets receives the new and removed targets and reconciles the current state.
// Any removals are removed from the allocator's targetItems and unassigned from the corresponding collector.
// Any net-new additions are assigned to the next available collector.
func (allocator *jobAverageAllocator) handleTargets(diff diff.Changes[*target.Item]) {
	// Check for removals
	for k, item := range allocator.targetItems {
		if _, ok := diff.Removals()[k]; ok {
			c := allocator.collectors[item.CollectorName]
			c.NumTargets--
			delete(allocator.targetItems, k)
			delete(allocator.targetItemsPerJobPerCollector[item.CollectorName][item.JobName], item.Hash())
			TargetsPerCollector.WithLabelValues(item.CollectorName, jobAverageStrategyName).Set(float64(c.NumTargets))

		}
	}

	// Check for additions
	for k, item := range diff.Additions() {
		// Do nothing if the item is already there
		if _, ok := allocator.targetItems[k]; ok {
			continue
		} else {
			// Add item to item pool and assign a collector
			allocator.addTargetToTargetItems(item)
		}
	}
}

// handleCollectors receives the new and removed collectors and reconciles the current state.
// Any removals are removed from the allocator's collectors. New collectors are added to the allocator's collector map.
// Finally, any targets of removed collectors are reallocated to the next available collector.
func (allocator *jobAverageAllocator) handleCollectors(diff diff.Changes[*Collector]) {
	// Clear removed collectors
	for _, k := range diff.Removals() {
		delete(allocator.collectors, k.Name)
		TargetsPerCollector.WithLabelValues(k.Name, jobAverageStrategyName).Set(0)
	}

	// Insert the new collectors
	for _, i := range diff.Additions() {
		allocator.collectors[i.Name] = NewCollector(i.Name, i.NodeName)
	}

	// re-Allocate all targets
	allocator.targetItemsPerJobPerCollector = make(map[string]map[string]map[string]bool)
	for _, k := range allocator.collectors {
		k.NumTargets = 0
	}
	for _, item := range allocator.targetItems {
		allocator.addTargetToTargetItems(item)
	}

}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (allocator *jobAverageAllocator) SetTargets(targets map[string]*target.Item) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetTargets", jobAverageStrategyName))
	defer timer.ObserveDuration()

	if allocator.filter != nil {
		targets = allocator.filter.Apply(targets)
	}
	RecordTargetsKept(targets)

	allocator.m.Lock()
	defer allocator.m.Unlock()

	if len(allocator.collectors) == 0 {
		allocator.log.Info("No collector instances present, saving targets to allocate to collector(s)")
		// If there were no targets discovered previously, assign this as the new set of target items
		if len(allocator.targetItems) == 0 {
			allocator.log.Info("Not discovered any targets previously, saving targets found to the targetItems set")
			for k, item := range targets {
				allocator.targetItems[k] = item
			}
		} else {
			// If there were previously discovered targets, add or remove accordingly
			targetsDiffEmptyCollectorSet := diff.Maps(allocator.targetItems, targets)

			// Check for additions
			if len(targetsDiffEmptyCollectorSet.Additions()) > 0 {
				allocator.log.Info("New targets discovered, adding new targets to the targetItems set")
				for k, item := range targetsDiffEmptyCollectorSet.Additions() {
					// Do nothing if the item is already there
					if _, ok := allocator.targetItems[k]; ok {
						continue
					} else {
						// Add item to item pool
						allocator.targetItems[k] = item
					}
				}
			}

			// Check for deletions
			if len(targetsDiffEmptyCollectorSet.Removals()) > 0 {
				allocator.log.Info("Targets removed, Removing targets from the targetItems set")
				for k := range targetsDiffEmptyCollectorSet.Removals() {
					// Delete item from target items
					delete(allocator.targetItems, k)
				}
			}
		}
		return
	}
	// Check for target changes
	targetsDiff := diff.Maps(allocator.targetItems, targets)
	// If there are any additions or removals
	if len(targetsDiff.Additions()) != 0 || len(targetsDiff.Removals()) != 0 {
		allocator.handleTargets(targetsDiff)
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (allocator *jobAverageAllocator) SetCollectors(collectors map[string]*Collector) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetCollectors", jobAverageStrategyName))
	defer timer.ObserveDuration()

	CollectorsAllocatable.WithLabelValues(jobAverageStrategyName).Set(float64(len(collectors)))
	if len(collectors) == 0 {
		allocator.log.Info("No collector instances present")
		return
	}

	allocator.m.Lock()
	defer allocator.m.Unlock()

	// Check for collector changes
	collectorsDiff := diff.Maps(allocator.collectors, collectors)
	if len(collectorsDiff.Additions()) != 0 || len(collectorsDiff.Removals()) != 0 {
		allocator.handleCollectors(collectorsDiff)
	}
}

func newJobAverageAllocator(log logr.Logger, opts ...AllocationOption) Allocator {
	lwAllocator := &jobAverageAllocator{
		log:                           log,
		collectors:                    make(map[string]*Collector),
		targetItems:                   make(map[string]*target.Item),
		targetItemsPerJobPerCollector: make(map[string]map[string]map[string]bool),
	}

	for _, opt := range opts {
		opt(lwAllocator)
	}

	return lwAllocator
}
