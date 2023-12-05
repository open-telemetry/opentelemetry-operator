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

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/diff"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

var _ Allocator = &perNodeAllocator{}

const (
	perNodeStrategyName = "per-node"

	nodeNameLabel model.LabelName = "__meta_kubernetes_pod_node_name"
)

type perNodeAllocator struct {
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

func (allocator *perNodeAllocator) handleCollectors(diff diff.Changes[*Collector]) {
	// Clear removed collectors
	for _, k := range diff.Removals() {
		delete(allocator.collectors, k.Name)
		delete(allocator.targetItemsPerJobPerCollector, k.Name)
		TargetsPerCollector.WithLabelValues(k.Name, perNodeStrategyName).Set(0)
	}

	// Insert the new collectors
	for _, i := range diff.Additions() {
		allocator.collectors[i.Name] = NewCollector(i.Name, i.Node)
	}
}

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
func (allocator *perNodeAllocator) handleTargets(diff diff.Changes[*target.Item]) {
	// Check for removals
	for k, item := range allocator.targetItems {
		// if the current item is in the removals list
		if _, ok := diff.Removals()[k]; ok {
			c := allocator.collectors[item.CollectorName]
			c.NumTargets--
			delete(allocator.targetItems, k)
			delete(allocator.targetItemsPerJobPerCollector[item.CollectorName][item.JobName], item.Hash())
			TargetsPerCollector.WithLabelValues(item.CollectorName, perNodeStrategyName).Set(float64(c.NumTargets))
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

func (allocator *perNodeAllocator) addTargetToTargetItems(tg *target.Item) {
	chosenCollector := allocator.findCollector(tg.Labels)
	// TODO: How to handle this edge case? Can we have items without a collector?
	if chosenCollector == nil {
		allocator.log.V(2).Info("Couldn't find a collector for the target item", "item", tg, "collectors", allocator.collectors)
		return
	}
	tg.CollectorName = chosenCollector.Name
	allocator.targetItems[tg.Hash()] = tg
	allocator.addCollectorTargetItemMapping(tg)
	chosenCollector.NumTargets++
	TargetsPerCollector.WithLabelValues(chosenCollector.Name, leastWeightedStrategyName).Set(float64(chosenCollector.NumTargets))
}

func (allocator *perNodeAllocator) findCollector(labels model.LabelSet) *Collector {
	var col *Collector
	for _, v := range allocator.collectors {
		if nodeNameLabelValue, ok := labels[nodeNameLabel]; ok {
			if v.Node == string(nodeNameLabelValue) {
				col = v
				break
			}
		}
	}

	return col
}

func (allocator *perNodeAllocator) addCollectorTargetItemMapping(tg *target.Item) {
	if allocator.targetItemsPerJobPerCollector[tg.CollectorName] == nil {
		allocator.targetItemsPerJobPerCollector[tg.CollectorName] = make(map[string]map[string]bool)
	}
	if allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] == nil {
		allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] = make(map[string]bool)
	}
	allocator.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName][tg.Hash()] = true
}

func (allocator *perNodeAllocator) TargetItems() map[string]*target.Item {
	allocator.m.RLock()
	defer allocator.m.RUnlock()
	targetItemsCopy := make(map[string]*target.Item)
	for k, v := range allocator.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

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
