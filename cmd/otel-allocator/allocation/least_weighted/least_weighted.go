package least_weighted

import (
	"os"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/strategy"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/utility"
)

func init() {
	err := strategy.Register("least-weighted", NewLeastWeightedStrategy)
	if err != nil {
		os.Exit(1)
	}
}

type LeastWeightedStrategy struct {
}

func NewLeastWeightedStrategy() strategy.Allocator {
	return &LeastWeightedStrategy{}
}

// findNextCollector finds the next collector with fewer number of targets.
// This method is called from within SetTargets and SetCollectors, whose caller
// acquires the needed lock. Requires there to be at least one collector set
func (l LeastWeightedStrategy) findNextCollector(state strategy.State) strategy.Collector {
	// Set a dummy to be replaced
	col := strategy.Collector{NumTargets: -1}
	for _, v := range state.Collectors() {
		if col.NumTargets == -1 || v.NumTargets < col.NumTargets {
			col = v
		}
	}
	return col
}

// addTargetToTargetItems assigns a target to the next available collector and adds it to the allocator's targetItems
// This method is called from within SetTargets and SetCollectors, whose caller acquires the needed lock.
// This is only called after the collectors are cleared or when a new target has been found in the tempTargetMap
func (l LeastWeightedStrategy) addTargetToTargetItems(target strategy.TargetItem, state strategy.State) strategy.State {
	nextState := state
	chosenCollector := l.findNextCollector(nextState)
	targetItem := strategy.NewTargetItem(target.JobName, target.TargetURL, target.Label, chosenCollector.Name)
	nextState = nextState.SetTargetItem(targetItem.Hash(), targetItem)
	chosenCollector.NumTargets++
	nextState = nextState.SetCollector(chosenCollector.Name, chosenCollector)
	strategy.TargetsPerCollector.WithLabelValues(chosenCollector.Name).Set(float64(chosenCollector.NumTargets))
	return nextState
}

func (l LeastWeightedStrategy) handleTargets(targetDiff utility.Changes[strategy.TargetItem], currentState strategy.State) strategy.State {
	nextState := currentState
	// Check for removals
	for k, target := range nextState.TargetItems() {
		// if the current target is in the removals list
		if _, ok := targetDiff.Removals()[k]; ok {
			c := nextState.Collectors()[target.CollectorName]
			c.NumTargets--
			nextState = nextState.SetCollector(target.CollectorName, c)
			nextState = nextState.RemoveTargetItem(k)
			strategy.TargetsPerCollector.WithLabelValues(target.CollectorName).Set(float64(nextState.Collectors()[target.CollectorName].NumTargets))
		}
	}

	// Check for additions
	for k, target := range targetDiff.Additions() {
		// Do nothing if the item is already there
		if _, ok := nextState.TargetItems()[k]; ok {
			continue
		} else {
			// Assign new set of collectors with the one different name
			nextState = l.addTargetToTargetItems(target, nextState)
		}
	}
	return nextState
}

func (l LeastWeightedStrategy) handleCollectors(collectorsDiff utility.Changes[strategy.Collector], currentState strategy.State) strategy.State {
	nextState := currentState
	// Clear existing collectors
	for _, k := range collectorsDiff.Removals() {
		nextState = nextState.RemoveCollector(k.Name)
		strategy.TargetsPerCollector.WithLabelValues(k.Name).Set(0)
	}
	// Insert the new collectors
	for _, i := range collectorsDiff.Additions() {
		nextState = nextState.SetCollector(i.Name, strategy.Collector{Name: i.Name, NumTargets: 0})
	}

	// find targets which need to be redistributed
	var redistribute []strategy.TargetItem
	for _, item := range nextState.TargetItems() {
		for _, s := range collectorsDiff.Removals() {
			if item.CollectorName == s.Name {
				redistribute = append(redistribute, item)
			}
		}
	}
	// Re-Allocate the existing targets
	for _, item := range redistribute {
		nextState = l.addTargetToTargetItems(item, nextState)
	}
	return nextState
}

func (l LeastWeightedStrategy) Allocate(currentState, newState strategy.State) strategy.State {
	nextState := currentState
	// Check for target changes
	targetsDiff := utility.DiffMaps(currentState.TargetItems(), newState.TargetItems())
	// If there are any additions or removals
	if len(targetsDiff.Additions()) != 0 || len(targetsDiff.Removals()) != 0 {
		nextState = l.handleTargets(targetsDiff, currentState)
	}
	// Check for collector changes
	collectorsDiff := utility.DiffMaps(currentState.Collectors(), newState.Collectors())
	// If there are any additions or removals
	if len(collectorsDiff.Additions()) != 0 || len(collectorsDiff.Removals()) != 0 {
		nextState = l.handleCollectors(collectorsDiff, nextState)
	}
	return nextState
}
