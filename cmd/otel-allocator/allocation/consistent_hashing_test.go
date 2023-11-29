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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanSetSingleTarget(t *testing.T) {
	cols := MakeNCollectors(3, 0)
	c := newConsistentHashingAllocator(logger)
	c.SetCollectors(cols)
	c.SetTargets(MakeNNewTargets(1, 3, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, 1)
	for _, item := range actualTargetItems {
		assert.Equal(t, "collector-0", item.CollectorName)
	}
}

func TestRelativelyEvenDistribution(t *testing.T) {
	numCols := 15
	numItems := 10000
	cols := MakeNCollectors(numCols, 0)
	var expectedPerCollector = float64(numItems / numCols)
	expectedDelta := (expectedPerCollector * 1.5) - expectedPerCollector
	c := newConsistentHashingAllocator(logger)
	c.SetCollectors(cols)
	c.SetTargets(MakeNNewTargets(numItems, 0, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, numItems)
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numCols)
	for _, col := range actualCollectors {
		assert.InDelta(t, col.NumTargets, expectedPerCollector, expectedDelta)
	}
}

func TestFullReallocation(t *testing.T) {
	cols := MakeNCollectors(10, 0)
	c := newConsistentHashingAllocator(logger)
	c.SetCollectors(cols)
	c.SetTargets(MakeNNewTargets(10000, 10, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, 10000)
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, 10)
	newCols := MakeNCollectors(10, 10)
	c.SetCollectors(newCols)
	updatedTargetItems := c.TargetItems()
	assert.Len(t, updatedTargetItems, 10000)
	updatedCollectors := c.Collectors()
	assert.Len(t, updatedCollectors, 10)
	for _, item := range updatedTargetItems {
		_, ok := updatedCollectors[item.CollectorName]
		assert.True(t, ok, "Some items weren't reallocated correctly")
	}
}

func TestNumRemapped(t *testing.T) {
	numItems := 10_000
	numInitialCols := 15
	numFinalCols := 16
	expectedDelta := float64((numFinalCols - numInitialCols) * (numItems / numFinalCols))
	cols := MakeNCollectors(numInitialCols, 0)
	c := newConsistentHashingAllocator(logger)
	c.SetCollectors(cols)
	c.SetTargets(MakeNNewTargets(numItems, numInitialCols, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, numItems)
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numInitialCols)
	newCols := MakeNCollectors(numFinalCols, 0)
	c.SetCollectors(newCols)
	updatedTargetItems := c.TargetItems()
	assert.Len(t, updatedTargetItems, numItems)
	updatedCollectors := c.Collectors()
	assert.Len(t, updatedCollectors, numFinalCols)
	countRemapped := 0
	countNotRemapped := 0
	for _, item := range updatedTargetItems {
		previousItem, ok := actualTargetItems[item.Hash()]
		assert.True(t, ok)
		if previousItem.CollectorName != item.CollectorName {
			countRemapped++
		} else {
			countNotRemapped++
		}
	}
	assert.InDelta(t, numItems/numFinalCols, countRemapped, expectedDelta)
}

func TestTargetsWithNoCollectorsConsistentHashing(t *testing.T) {

	c := newConsistentHashingAllocator(logger)

	// Adding 10 new targets
	numItems := 10
	c.SetTargets(MakeNNewTargetsWithEmptyCollectors(numItems, 0))
	actualTargetItems := c.TargetItems()
	assert.Len(t, actualTargetItems, numItems)

	// Adding 5 new targets, and removing the old 10 targets
	numItemsUpdate := 5
	c.SetTargets(MakeNNewTargetsWithEmptyCollectors(numItemsUpdate, 10))
	actualTargetItemsUpdated := c.TargetItems()
	assert.Len(t, actualTargetItemsUpdated, numItemsUpdate)

	// Adding 5 new targets, and one existing target
	numItemsUpdate = 6
	c.SetTargets(MakeNNewTargetsWithEmptyCollectors(numItemsUpdate, 14))
	actualTargetItemsUpdated = c.TargetItems()
	assert.Len(t, actualTargetItemsUpdated, numItemsUpdate)

	// Adding collectors to test allocation
	numCols := 2
	cols := MakeNCollectors(2, 0)
	c.SetCollectors(cols)
	var expectedPerCollector = float64(numItemsUpdate / numCols)
	expectedDelta := (expectedPerCollector * 1.5) - expectedPerCollector
	// Checking to see that there is no change to number of targets
	actualTargetItems = c.TargetItems()
	assert.Len(t, actualTargetItems, numItemsUpdate)
	// Checking to see collectors are added correctly
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numCols)
	for _, col := range actualCollectors {
		assert.InDelta(t, col.NumTargets, expectedPerCollector, expectedDelta)
	}
}
