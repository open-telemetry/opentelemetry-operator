package allocation

var _ AllocatorStrategy = LeastWeightedStrategy{}

type LeastWeightedStrategy struct {
}

func NewLeastWeightedStrategy() *LeastWeightedStrategy {
	return &LeastWeightedStrategy{}
}

// findNextCollector finds the next collector with fewer number of targets.
// This method is called from within SetTargets and SetCollectors, whose caller
// acquires the needed lock. Requires there to be at least one collector set
func (l LeastWeightedStrategy) findNextCollector(state State) collector {
	// Set a dummy to be replaced
	col := collector{NumTargets: -1}
	for _, v := range state.collectors {
		if col.NumTargets == -1 || v.NumTargets < col.NumTargets {
			col = v
		}
	}
	return col
}

// addTargetToTargetItems assigns a target to the next available collector and adds it to the allocator's targetItems
// This method is called from within SetTargets and SetCollectors, whose caller acquires the needed lock.
// This is only called after the collectors are cleared or when a new target has been found in the tempTargetMap
func (l LeastWeightedStrategy) addTargetToTargetItems(target TargetItem, state State) State {
	nextState := state
	chosenCollector := l.findNextCollector(nextState)
	targetItem := NewTargetItem(target.JobName, target.TargetURL, target.Label, chosenCollector.Name)
	nextState.targetItems[targetItem.hash()] = targetItem
	chosenCollector.NumTargets++
	nextState.collectors[chosenCollector.Name] = chosenCollector
	targetsPerCollector.WithLabelValues(chosenCollector.Name).Set(float64(chosenCollector.NumTargets))
	return nextState
}

func (l LeastWeightedStrategy) handleTargets(targetDiff changes[TargetItem], currentState State) State {
	nextState := currentState
	// Check for removals
	for k, target := range nextState.targetItems {
		// if the current target is in the removals list
		if _, ok := targetDiff.removals[k]; ok {
			c := nextState.collectors[target.CollectorName]
			c.NumTargets--
			nextState.collectors[target.CollectorName] = c
			delete(nextState.targetItems, k)
			targetsPerCollector.WithLabelValues(target.CollectorName).Set(float64(nextState.collectors[target.CollectorName].NumTargets))
		}
	}

	// Check for additions
	for k, target := range targetDiff.additions {
		// Do nothing if the item is already there
		if _, ok := nextState.targetItems[k]; ok {
			continue
		} else {
			// Assign new set of collectors with the one different name
			nextState = l.addTargetToTargetItems(target, nextState)
		}
	}
	return nextState
}

func (l LeastWeightedStrategy) handleCollectors(collectorsDiff changes[collector], currentState State) State {
	nextState := currentState
	// Clear existing collectors
	for _, k := range collectorsDiff.removals {
		delete(nextState.collectors, k.Name)
		targetsPerCollector.WithLabelValues(k.Name).Set(0)
	}
	// Insert the new collectors
	for _, i := range collectorsDiff.additions {
		nextState.collectors[i.Name] = collector{Name: i.Name, NumTargets: 0}
	}

	// find targets which need to be redistributed
	var redistribute []TargetItem
	for _, item := range nextState.targetItems {
		for _, s := range collectorsDiff.removals {
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

func (l LeastWeightedStrategy) Allocate(currentState, newState State) State {
	nextState := currentState
	// Check for target changes
	targetsDiff := diff(currentState.targetItems, newState.targetItems)
	// If there are any additions or removals
	if len(targetsDiff.additions) != 0 || len(targetsDiff.removals) != 0 {
		nextState = l.handleTargets(targetsDiff, currentState)
	}
	// Check for collector changes
	collectorsDiff := diff(currentState.collectors, newState.collectors)
	// If there are any additions or removals
	if len(collectorsDiff.additions) != 0 || len(collectorsDiff.removals) != 0 {
		nextState = l.handleCollectors(collectorsDiff, nextState)
	}
	return nextState
}
