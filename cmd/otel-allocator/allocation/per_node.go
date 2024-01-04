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

var _ Allocator = &perNodeAllocator{}

const perNodeStrategyName = "per-node"

// perNodeAllocator makes decisions to distribute work among
// a number of OpenTelemetry collectors based on the node on which
// the collector is running. This allocator should be used only when
// collectors are running as daemon set (agent) on each node.
// Users need to call SetTargets when they have new targets in their
// clusters and call SetCollectors when the collectors have changed.
type perNodeAllocator struct {
	// m protects collectors and targetItems for concurrent use.
	m sync.RWMutex
	// collectors is a map from a Collector's node name to a Collector instance
	collectors map[string]*Collector
	// targetItems is a map from a target item's hash to the target items allocated state
	targetItems map[string]*target.Item

	// collectorKey -> job -> target item hash -> true
	targetItemsPerJobPerCollector map[string]map[string]map[string]bool

	log logr.Logger

	filter Filter
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (allocator *perNodeAllocator) SetCollectors(collectors map[string]*Collector) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetCollectors", perNodeStrategyName))
	defer timer.ObserveDuration()

	CollectorsAllocatable.WithLabelValues(perNodeStrategyName).Set(float64(len(collectors)))
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

// handleCollectors receives the new and removed collectors and reconciles the current state.
// Any removals are removed from the allocator's collectors. New collectors are added to the allocator's collector map.
func (allocator *perNodeAllocator) handleCollectors(diff diff.Changes[*Collector]) {
	// Clear removed collectors
	for _, k := range diff.Removals() {
		delete(allocator.collectors, k.NodeName)
		delete(allocator.targetItemsPerJobPerCollector, k.Name)
		TargetsPerCollector.WithLabelValues(k.Name, perNodeStrategyName).Set(0)
	}

	// Insert the new collectors
	for _, i := range diff.Additions() {
		allocator.collectors[i.NodeName] = NewCollector(i.Name, i.NodeName)
	}

	// For a case where a collector is removed and added back, we need
	// to re-allocate any already existing targets.
	for _, item := range allocator.targetItems {
		allocator.addTargetToTargetItems(item)
	}
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (allocator *perNodeAllocator) SetTargets(targets map[string]*target.Item) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetTargets", perNodeStrategyName))
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

// handleTargets receives the new and removed targets and reconciles the current state.
// Any removals are removed from the allocator's targetItems and unassigned from the corresponding collector.
// Any net-new additions are assigned to the collector on the same node as the target.
func (allocator *perNodeAllocator) handleTargets(diff diff.Changes[*target.Item]) {
	// Check for removals
	for k, item := range allocator.targetItems {
		// if the current item is in the removals list
		if _, ok := diff.Removals()[k]; ok {
			c, ok := allocator.collectors[item.GetNodeName()]
			if !ok {
				continue
			}
			c.NumTargets--
			delete(allocator.targetItems, k)
			delete(allocator.targetItemsPerJobPerCollector[item.CollectorName][item.JobName], item.Hash())
			TargetsPerCollector.WithLabelValues(item.CollectorName, perNodeStrategyName).Set(float64(c.NumTargets))
		}
	}

	// Check for additions
	unassignedTargetsForJobs := make(map[string]struct{})
	for k, item := range diff.Additions() {
		// Do nothing if the item is already there
		if _, ok := allocator.targetItems[k]; ok {
			continue
		} else {
			// Add item to item pool and assign a collector
			collectorAssigned := allocator.addTargetToTargetItems(item)
			if !collectorAssigned {
				unassignedTargetsForJobs[item.JobName] = struct{}{}
			}
		}
	}

	// Check for unassigned targets
	if len(unassignedTargetsForJobs) > 0 {
		jobs := make([]string, 0, len(unassignedTargetsForJobs))
		for j := range unassignedTargetsForJobs {
			jobs = append(jobs, j)
		}

		allocator.log.Info("Could not assign targets for the following jobs due to missing node labels", "jobs", jobs)
	}
}

// addTargetToTargetItems assigns a target to the  collector and adds it to the allocator's targetItems
// This method is called from within SetTargets and SetCollectors, which acquire the needed lock.
// This is only called after the collectors are cleared or when a new target has been found in the tempTargetMap.
// INVARIANT: allocator.collectors must have at least 1 collector set.
// NOTE: by not creating a new target item, there is the potential for a race condition where we modify this target
// item while it's being encoded by the server JSON handler.
// Also, any targets that cannot be assigned to a collector, due to no matching node name, will remain unassigned. These
// targets are still "silently" added to the targetItems map, to prevent them from being reported as unassigned on each new
// target items setting.
func (allocator *perNodeAllocator) addTargetToTargetItems(tg *target.Item) bool {
	allocator.targetItems[tg.Hash()] = tg
	chosenCollector, ok := allocator.collectors[tg.GetNodeName()]
	if !ok {
		allocator.log.V(2).Info("Couldn't find a collector for the target item", "item", tg, "collectors", allocator.collectors)
		return false
	}
	tg.CollectorName = chosenCollector.Name
	allocator.addCollectorTargetItemMapping(tg)
	chosenCollector.NumTargets++
	TargetsPerCollector.WithLabelValues(chosenCollector.Name, perNodeStrategyName).Set(float64(chosenCollector.NumTargets))
	return true
}

// addCollectorTargetItemMapping keeps track of which collector has which jobs and targets
// this allows the allocator to respond without any extra allocations to http calls. The caller of this method
// has to acquire a lock.
func (allocator *perNodeAllocator) addCollectorTargetItemMapping(tg *target.Item) {
	if allocator.targetItemsPerJobPerCollector[tg.CollectorName] == nil {
		allocator.targetItemsPerJobPerCollector[tg.CollectorName] = make(map[string]map[string]bool)
	}
	if allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] == nil {
		allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] = make(map[string]bool)
	}
	allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName][tg.Hash()] = true
}

// TargetItems returns a shallow copy of the targetItems map.
func (allocator *perNodeAllocator) TargetItems() map[string]*target.Item {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]*target.Item)
	for k, v := range allocator.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (allocator *perNodeAllocator) Collectors() map[string]*Collector {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	collectorsCopy := make(map[string]*Collector)
	for k, v := range allocator.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

func (allocator *perNodeAllocator) GetTargetsForCollectorAndJob(collector string, job string) []*target.Item {
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

// SetFilter sets the filtering hook to use.
func (allocator *perNodeAllocator) SetFilter(filter Filter) {
	allocator.filter = filter
}

func newPerNodeAllocator(log logr.Logger, opts ...AllocationOption) Allocator {
	pnAllocator := &perNodeAllocator{
		log:                           log,
		collectors:                    make(map[string]*Collector),
		targetItems:                   make(map[string]*target.Item),
		targetItemsPerJobPerCollector: make(map[string]map[string]map[string]bool),
	}

	for _, opt := range opts {
		opt(pnAllocator)
	}

	return pnAllocator
}
