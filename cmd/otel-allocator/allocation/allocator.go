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
	"errors"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/diff"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

/*
	Target Allocator will serve on an HTTP server exposing /jobs/<job_id>/targets
	The targets are allocated using the least connection method
	Target Allocator will need information about the collectors in order to set the URLs
	Keep a Map of what each collector currently holds and update it based on new scrape target updates
*/

type Strategy interface {
	GetCollectorForTarget(map[string]*Collector, *target.Item) (*Collector, error)
	SetCollectors(map[string]*Collector)
	GetName() string
}

var _ Allocator = &TargetAllocator{}

func newAllocator(log logr.Logger, strategy Strategy, opts ...AllocationOption) Allocator {
	chAllocator := &TargetAllocator{
		strategy:                      strategy,
		collectors:                    make(map[string]*Collector),
		targetItems:                   make(map[string]*target.Item),
		targetItemsPerJobPerCollector: make(map[string]map[string]map[string]bool),
		log:                           log,
	}
	for _, opt := range opts {
		opt(chAllocator)
	}

	return chAllocator
}

type TargetAllocator struct {
	strategy Strategy
	// m protects consistentHasher, collectors and targetItems for concurrent use.
	m sync.RWMutex

	// collectors is a map from a Collector's name to a Collector instance
	// collectorKey -> collector pointer
	collectors map[string]*Collector

	// targetItems is a map from a target item's hash to the target items allocated state
	// targetItem hash -> target item pointer
	targetItems map[string]*target.Item

	// collectorKey -> job -> target item hash -> true
	targetItemsPerJobPerCollector map[string]map[string]map[string]bool

	log logr.Logger

	filter Filter
}

// SetFilter sets the filtering hook to use.
func (t *TargetAllocator) SetFilter(filter Filter) {
	t.filter = filter
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (t *TargetAllocator) SetTargets(targets map[string]*target.Item) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetTargets", perNodeStrategyName))
	defer timer.ObserveDuration()

	if t.filter != nil {
		targets = t.filter.Apply(targets)
	}
	RecordTargetsKept(targets)

	t.m.Lock()
	defer t.m.Unlock()

	// Check for target changes
	targetsDiff := diff.Maps(t.targetItems, targets)
	// If there are any additions or removals
	if len(targetsDiff.Additions()) != 0 || len(targetsDiff.Removals()) != 0 {
		t.handleTargets(targetsDiff)
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (t *TargetAllocator) SetCollectors(collectors map[string]*Collector) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetCollectors", consistentHashingStrategyName))
	defer timer.ObserveDuration()

	CollectorsAllocatable.WithLabelValues(consistentHashingStrategyName).Set(float64(len(collectors)))
	if len(collectors) == 0 {
		t.log.Info("No collector instances present")
		return
	}

	t.m.Lock()
	defer t.m.Unlock()

	// Check for collector changes
	collectorsDiff := diff.Maps(t.collectors, collectors)
	if len(collectorsDiff.Additions()) != 0 || len(collectorsDiff.Removals()) != 0 {
		t.handleCollectors(collectorsDiff)
	}
}

func (t *TargetAllocator) GetTargetsForCollectorAndJob(collector string, job string) []*target.Item {
	t.m.RLock()
	defer t.m.RUnlock()
	if _, ok := t.targetItemsPerJobPerCollector[collector]; !ok {
		return []*target.Item{}
	}
	if _, ok := t.targetItemsPerJobPerCollector[collector][job]; !ok {
		return []*target.Item{}
	}
	targetItemsCopy := make([]*target.Item, len(t.targetItemsPerJobPerCollector[collector][job]))
	index := 0
	for targetHash := range t.targetItemsPerJobPerCollector[collector][job] {
		targetItemsCopy[index] = t.targetItems[targetHash]
		index++
	}
	return targetItemsCopy
}

// TargetItems returns a shallow copy of the targetItems map.
func (t *TargetAllocator) TargetItems() map[string]*target.Item {
	t.m.RLock()
	defer t.m.RUnlock()
	targetItemsCopy := make(map[string]*target.Item)
	for k, v := range t.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (t *TargetAllocator) Collectors() map[string]*Collector {
	t.m.RLock()
	defer t.m.RUnlock()
	collectorsCopy := make(map[string]*Collector)
	for k, v := range t.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// handleTargets receives the new and removed targets and reconciles the current state.
// Any removals are removed from the allocator's targetItems and unassigned from the corresponding collector.
// Any net-new additions are assigned to the collector on the same node as the target.
func (t *TargetAllocator) handleTargets(diff diff.Changes[*target.Item]) {
	// Check for removals
	for k, item := range t.targetItems {
		// if the current item is in the removals list
		if _, ok := diff.Removals()[k]; ok {
			c, ok := t.collectors[item.CollectorName]
			if ok {
				c.NumTargets--
				TargetsPerCollector.WithLabelValues(item.CollectorName, perNodeStrategyName).Set(float64(c.NumTargets))
			}
			delete(t.targetItems, k)
			delete(t.targetItemsPerJobPerCollector[item.CollectorName][item.JobName], item.Hash())
		}
	}

	// Check for additions
	assignmentErrors := []error{}
	for k, item := range diff.Additions() {
		// Do nothing if the item is already there
		if _, ok := t.targetItems[k]; ok {
			continue
		} else {
			item.CollectorName = ""
			// Add item to item pool and assign a collector
			err := t.addTargetToTargetItems(item)
			if err != nil {
				assignmentErrors = append(assignmentErrors, err)
			}
		}
	}

	// Check for unassigned targets
	unassignedTargets := len(assignmentErrors)
	if unassignedTargets > 0 {
		err := errors.Join(assignmentErrors...)
		t.log.Info("Could not assign targets for some jobs due to missing node labels", "targets", unassignedTargets, "error", err)
		TargetsUnassigned.Set(float64(unassignedTargets))
	}
}

func (t *TargetAllocator) addTargetToTargetItems(tg *target.Item) error {
	// Short-circuit if there's no collectors
	// Check if this is a reassignment, if so, decrement the previous collector's NumTargets
	if previousColName, ok := t.collectors[tg.CollectorName]; ok {
		previousColName.NumTargets--
		delete(t.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName], tg.Hash())
		TargetsPerCollector.WithLabelValues(previousColName.String(), consistentHashingStrategyName).Set(float64(t.collectors[previousColName.String()].NumTargets))
	}
	t.targetItems[tg.Hash()] = tg
	if len(t.collectors) > 0 {
		colOwner, err := t.strategy.GetCollectorForTarget(t.collectors, tg)
		if err != nil {
			return err
		}
		tg.CollectorName = colOwner.Name
		t.addCollectorTargetItemMapping(tg)
		t.collectors[colOwner.String()].NumTargets++
		TargetsPerCollector.WithLabelValues(colOwner.String(), consistentHashingStrategyName).Set(float64(t.collectors[colOwner.String()].NumTargets))
	}
	return nil
}

// addCollectorTargetItemMapping keeps track of which collector has which jobs and targets
// this allows the allocator to respond without any extra allocations to http calls. The caller of this method
// has to acquire a lock.
func (t *TargetAllocator) addCollectorTargetItemMapping(tg *target.Item) {
	if t.targetItemsPerJobPerCollector[tg.CollectorName] == nil {
		t.targetItemsPerJobPerCollector[tg.CollectorName] = make(map[string]map[string]bool)
	}
	if t.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] == nil {
		t.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] = make(map[string]bool)
	}
	t.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName][tg.Hash()] = true
}

// handleCollectors receives the new and removed collectors and reconciles the current state.
// Any removals are removed from the allocator's collectors. New collectors are added to the allocator's collector map.
// Finally, update all targets' collectors to match the consistent hashing.
func (t *TargetAllocator) handleCollectors(diff diff.Changes[*Collector]) {
	// Clear removed collectors
	for _, k := range diff.Removals() {
		delete(t.collectors, k.Name)
		delete(t.targetItemsPerJobPerCollector, k.Name)
		TargetsPerCollector.WithLabelValues(k.Name, consistentHashingStrategyName).Set(0)
	}
	// Insert the new collectors
	for _, i := range diff.Additions() {
		t.collectors[i.Name] = NewCollector(i.Name, i.NodeName)
	}

	// Set collectors on the strategy
	t.strategy.SetCollectors(t.collectors)

	// Re-Allocate all targets
	assignmentErrors := []error{}
	for _, item := range t.targetItems {
		err := t.addTargetToTargetItems(item)
		if err != nil {
			assignmentErrors = append(assignmentErrors, err)
		}
	}
	// Check for unassigned targets
	unassignedTargets := len(assignmentErrors)
	if unassignedTargets > 0 {
		err := errors.Join(assignmentErrors...)
		t.log.Info("Could not assign targets for some jobs due to missing node labels", "targets", unassignedTargets, "error", err)
		TargetsUnassigned.Set(float64(unassignedTargets))
	}
}
