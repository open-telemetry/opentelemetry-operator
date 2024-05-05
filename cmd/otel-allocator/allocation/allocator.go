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

var _ Allocator = &allocator{}

func newAllocator(log logr.Logger, strategy Strategy, opts ...AllocationOption) Allocator {
	chAllocator := &allocator{
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

type allocator struct {
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
func (a *allocator) SetFilter(filter Filter) {
	a.filter = filter
}

// SetTargets accepts a list of targets that will be used to make
// load balancing decisions. This method should be called when there are
// new targets discovered or existing targets are shutdown.
func (a *allocator) SetTargets(targets map[string]*target.Item) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetTargets", a.strategy.GetName()))
	defer timer.ObserveDuration()

	if a.filter != nil {
		targets = a.filter.Apply(targets)
	}
	RecordTargetsKept(targets)

	a.m.Lock()
	defer a.m.Unlock()

	// Check for target changes
	targetsDiff := diff.Maps(a.targetItems, targets)
	// If there are any additions or removals
	if len(targetsDiff.Additions()) != 0 || len(targetsDiff.Removals()) != 0 {
		a.handleTargets(targetsDiff)
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
// This method is called when Collectors are added or removed.
func (a *allocator) SetCollectors(collectors map[string]*Collector) {
	timer := prometheus.NewTimer(TimeToAssign.WithLabelValues("SetCollectors", a.strategy.GetName()))
	defer timer.ObserveDuration()

	CollectorsAllocatable.WithLabelValues(a.strategy.GetName()).Set(float64(len(collectors)))
	if len(collectors) == 0 {
		a.log.Info("No collector instances present")
		return
	}

	a.m.Lock()
	defer a.m.Unlock()

	// Check for collector changes
	collectorsDiff := diff.Maps(a.collectors, collectors)
	if len(collectorsDiff.Additions()) != 0 || len(collectorsDiff.Removals()) != 0 {
		a.handleCollectors(collectorsDiff)
	}
}

func (a *allocator) GetTargetsForCollectorAndJob(collector string, job string) []*target.Item {
	a.m.RLock()
	defer a.m.RUnlock()
	if _, ok := a.targetItemsPerJobPerCollector[collector]; !ok {
		return []*target.Item{}
	}
	if _, ok := a.targetItemsPerJobPerCollector[collector][job]; !ok {
		return []*target.Item{}
	}
	targetItemsCopy := make([]*target.Item, len(a.targetItemsPerJobPerCollector[collector][job]))
	index := 0
	for targetHash := range a.targetItemsPerJobPerCollector[collector][job] {
		targetItemsCopy[index] = a.targetItems[targetHash]
		index++
	}
	return targetItemsCopy
}

// TargetItems returns a shallow copy of the targetItems map.
func (a *allocator) TargetItems() map[string]*target.Item {
	a.m.RLock()
	defer a.m.RUnlock()
	targetItemsCopy := make(map[string]*target.Item)
	for k, v := range a.targetItems {
		targetItemsCopy[k] = v
	}
	return targetItemsCopy
}

// Collectors returns a shallow copy of the collectors map.
func (a *allocator) Collectors() map[string]*Collector {
	a.m.RLock()
	defer a.m.RUnlock()
	collectorsCopy := make(map[string]*Collector)
	for k, v := range a.collectors {
		collectorsCopy[k] = v
	}
	return collectorsCopy
}

// handleTargets receives the new and removed targets and reconciles the current state.
// Any removals are removed from the allocator's targetItems and unassigned from the corresponding collector.
// Any net-new additions are assigned to the collector on the same node as the target.
func (a *allocator) handleTargets(diff diff.Changes[*target.Item]) {
	// Check for removals
	for k, item := range a.targetItems {
		// if the current item is in the removals list
		if _, ok := diff.Removals()[k]; ok {
			c, ok := a.collectors[item.CollectorName]
			if ok {
				c.NumTargets--
				TargetsPerCollector.WithLabelValues(item.CollectorName, a.strategy.GetName()).Set(float64(c.NumTargets))
			}
			delete(a.targetItems, k)
			delete(a.targetItemsPerJobPerCollector[item.CollectorName][item.JobName], item.Hash())
		}
	}

	// Check for additions
	assignmentErrors := []error{}
	for k, item := range diff.Additions() {
		// Do nothing if the item is already there
		if _, ok := a.targetItems[k]; ok {
			continue
		} else {
			// TODO: track target -> collector relationship in a separate map
			item.CollectorName = ""
			// Add item to item pool and assign a collector
			err := a.addTargetToTargetItems(item)
			if err != nil {
				assignmentErrors = append(assignmentErrors, err)
			}
		}
	}

	// Check for unassigned targets
	unassignedTargets := len(assignmentErrors)
	if unassignedTargets > 0 {
		err := errors.Join(assignmentErrors...)
		a.log.Info("Could not assign targets for some jobs due to missing node labels", "targets", unassignedTargets, "error", err)
		TargetsUnassigned.Set(float64(unassignedTargets))
	}
}

func (a *allocator) addTargetToTargetItems(tg *target.Item) error {
	// Check if this is a reassignment, if so, decrement the previous collector's NumTargets
	if previousColName, ok := a.collectors[tg.CollectorName]; ok {
		previousColName.NumTargets--
		delete(a.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName], tg.Hash())
		TargetsPerCollector.WithLabelValues(previousColName.String(), a.strategy.GetName()).Set(float64(a.collectors[previousColName.String()].NumTargets))
	}
	a.targetItems[tg.Hash()] = tg
	if len(a.collectors) > 0 {
		colOwner, err := a.strategy.GetCollectorForTarget(a.collectors, tg)
		if err != nil {
			return err
		}
		tg.CollectorName = colOwner.Name
		a.addCollectorTargetItemMapping(tg)
		a.collectors[colOwner.String()].NumTargets++
		TargetsPerCollector.WithLabelValues(colOwner.String(), a.strategy.GetName()).Set(float64(a.collectors[colOwner.String()].NumTargets))
	}
	return nil
}

// addCollectorTargetItemMapping keeps track of which collector has which jobs and targets
// this allows the allocator to respond without any extra allocations to http calls. The caller of this method
// has to acquire a lock.
func (a *allocator) addCollectorTargetItemMapping(tg *target.Item) {
	if a.targetItemsPerJobPerCollector[tg.CollectorName] == nil {
		a.targetItemsPerJobPerCollector[tg.CollectorName] = make(map[string]map[string]bool)
	}
	if a.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] == nil {
		a.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName] = make(map[string]bool)
	}
	a.targetItemsPerJobPerCollector[tg.CollectorName][tg.JobName][tg.Hash()] = true
}

// handleCollectors receives the new and removed collectors and reconciles the current state.
// Any removals are removed from the allocator's collectors. New collectors are added to the allocator's collector map.
// Finally, update all targets' collectors to match the consistent hashing.
func (a *allocator) handleCollectors(diff diff.Changes[*Collector]) {
	// Clear removed collectors
	for _, k := range diff.Removals() {
		delete(a.collectors, k.Name)
		delete(a.targetItemsPerJobPerCollector, k.Name)
		TargetsPerCollector.WithLabelValues(k.Name, a.strategy.GetName()).Set(0)
	}
	// Insert the new collectors
	for _, i := range diff.Additions() {
		a.collectors[i.Name] = NewCollector(i.Name, i.NodeName)
	}

	// Set collectors on the strategy
	a.strategy.SetCollectors(a.collectors)

	// Re-Allocate all targets
	assignmentErrors := []error{}
	for _, item := range a.targetItems {
		err := a.addTargetToTargetItems(item)
		if err != nil {
			assignmentErrors = append(assignmentErrors, err)
		}
	}
	// Check for unassigned targets
	unassignedTargets := len(assignmentErrors)
	if unassignedTargets > 0 {
		err := errors.Join(assignmentErrors...)
		a.log.Info("Could not assign targets for some jobs due to missing node labels", "targets", unassignedTargets, "error", err)
		TargetsUnassigned.Set(float64(unassignedTargets))
	}
}
