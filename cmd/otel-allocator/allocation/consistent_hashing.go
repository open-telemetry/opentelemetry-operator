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

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash/v2"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/diff"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var _ Allocator = &consistentHashingAllocator{}

const consistentHashingStrategyName = "consistent-hashing"

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

type consistentHashingAllocator struct {
	// m protects consistentHasher, collectors and targetItems for concurrent use.
	m sync.RWMutex

	consistentHasher *consistent.Consistent

	// collectors is a map from a Collector's name to a Collector instance
	collectors map[string]*Collector

	// targetItems is a map from a target item's hash to the target items allocated state
	targetItems map[string]*target.Item

	log logr.Logger

	filter Filter
}

func newConsistentHashingAllocator(log logr.Logger, opts ...AllocationOption) Allocator {
	config := consistent.Config{
		PartitionCount:    1061,
		ReplicationFactor: 5,
		Load:              1.1,
		Hasher:            hasher{},
	}
	consistentHasher := consistent.New(nil, config)
	chAllocator := &consistentHashingAllocator{
		consistentHasher: consistentHasher,
		collectors:       make(map[string]*Collector),
		targetItems:      make(map[string]*target.Item),
		log:              log,
	}
	for _, opt := range opts {
		opt(chAllocator)
	}

	return chAllocator
}

// SetFilter sets the filtering hook to use.
func (c *consistentHashingAllocator) SetFilter(filter Filter) {
	c.filter = filter
}

// addTargetToTargetItems assigns a target to the collector based on its hash and adds it to the allocator's targetItems
// This method is called from within SetTargets and SetCollectors, which acquire the needed lock.
// This is only called after the collectors are cleared or when a new target has been found in the tempTargetMap.
// INVARIANT: c.collectors must have at least 1 collector set.
func (c *consistentHashingAllocator) addTargetToTargetItems(tg *target.Item) {
	// Check if this is a reassignment, if so, decrement the previous collector's NumTargets
	if previousColName, ok := c.collectors[tg.CollectorName]; ok {
		previousColName.NumTargets--
		TargetsPerCollector.WithLabelValues(previousColName.String(), consistentHashingStrategyName).Set(float64(c.collectors[previousColName.String()].NumTargets))
	}
	colOwner := c.consistentHasher.LocateKey([]byte(tg.Hash()))
	targetItem := target.NewItem(tg.JobName, tg.TargetURL, tg.Label, colOwner.String())
	c.targetItems[targetItem.Hash()] = targetItem
	c.collectors[colOwner.String()].NumTargets++
	TargetsPerCollector.WithLabelValues(colOwner.String(), consistentHashingStrategyName).Set(float64(c.collectors[colOwner.String()].NumTargets))
}

// handleTargets receives the new and removed targets and reconciles the current state.
// Any removals are removed from the allocator's targetItems and unassigned from the corresponding collector.
// Any net-new additions are assigned to the next available collector.
func (c *consistentHashingAllocator) handleTargets(diff diff.Changes[*target.Item]) {
	// Check for removals
	for k, target := range c.targetItems {
		// if the current target is in the removals list
		if _, ok := diff.Removals()[k]; ok {
			col := c.collectors[target.CollectorName]
			col.NumTargets--
			delete(c.targetItems, k)
			TargetsPerCollector.WithLabelValues(target.CollectorName, consistentHashingStrategyName).Set(float64(col.NumTargets))
		}
	}

	// Check for additions
	for k, target := range diff.Additions() {
		// Do nothing if the item is already there
		if _, ok := c.targetItems[k]; ok {
			continue
		} else {
			// Add target to target pool and assign a collector
			c.addTargetToTargetItems(target)
		}
	}
}

// handleCollectors receives the new and removed collectors and reconciles the current state.
// Any removals are removed from the allocator's collectors. New collectors are added to the allocator's collector map.
// Finally, update all targets' collectors to match the consistent hashing.
func (c *consistentHashingAllocator) handleCollectors(diff diff.Changes[*Collector]) {
	// Clear removed collectors
	for _, k := range diff.Removals() {
		delete(c.collectors, k.Name)
		c.consistentHasher.Remove(k.Name)
		TargetsPerCollector.WithLabelValues(k.Name, consistentHashingStrategyName).Set(0)
	}
	// Insert the new collectors
	for _, i := range diff.Additions() {
		c.collectors[i.Name] = NewCollector(i.Name)
		c.consistentHasher.Add(c.collectors[i.Name])
	}

	// Re-Allocate all targets
	for _, item := range c.targetItems {
		c.addTargetToTargetItems(item)
	}
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (c *consistentHashingAllocator) SetTargets(targets map[string]*target.Item) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetTargets", consistentHashingStrategyName))
	defer timer.ObserveDuration()

	if c.filter != nil {
		targets = c.filter.Apply(targets)
	}
	RecordTargetsKeptPerJob(targets)

	c.m.Lock()
	defer c.m.Unlock()

	if len(c.collectors) == 0 {
		c.log.Info("No collector instances present, cannot set targets")
		return
	}
	// Check for target changes
	targetsDiff := diff.Maps(c.targetItems, targets)
	// If there are any additions or removals
	if len(targetsDiff.Additions()) != 0 || len(targetsDiff.Removals()) != 0 {
		c.handleTargets(targetsDiff)
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (c *consistentHashingAllocator) SetCollectors(collectors map[string]*Collector) {
	log := c.log.WithValues("component", "opentelemetry-targetallocator")
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetCollectors", consistentHashingStrategyName))
	defer timer.ObserveDuration()

	CollectorsAllocatable.WithLabelValues(consistentHashingStrategyName).Set(float64(len(collectors)))
	if len(collectors) == 0 {
		log.Info("No collector instances present")
		return
	}

	c.m.Lock()
	defer c.m.Unlock()

	// Check for collector changes
	collectorsDiff := diff.Maps(c.collectors, collectors)
	if len(collectorsDiff.Additions()) != 0 || len(collectorsDiff.Removals()) != 0 {
		c.handleCollectors(collectorsDiff)
	}
}

// TargetItems returns a shallow copy of the targetItems map.
func (c *consistentHashingAllocator) TargetItems() map[string]*target.Item {
	c.m.RLock()
	defer c.m.RUnlock()
	targetItemsCopy := make(map[string]*target.Item)
	for k, v := range c.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (c *consistentHashingAllocator) Collectors() map[string]*Collector {
	c.m.RLock()
	defer c.m.RUnlock()
	collectorsCopy := make(map[string]*Collector)
	for k, v := range c.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}
