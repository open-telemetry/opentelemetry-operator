// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

func TestNoCollectorReassignment(t *testing.T) {
	s, _ := New("least-weighted", logger)

	cols := MakeNCollectors(3, 0)
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	assert.Len(t, s.Collectors(), expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.Collectors()[i.Name])
	}
	initTargets := MakeNNewTargets(6, 3, 0)

	// test that targets and collectors are added properly
	s.SetTargets(initTargets)

	// verify
	expectedTargetLen := len(initTargets)
	targetItems := s.TargetItems()
	assert.Len(t, targetItems, expectedTargetLen)

	// assign new set of collectors with the same names
	newCols := MakeNCollectors(3, 0)
	s.SetCollectors(newCols)

	newTargetItems := s.TargetItems()
	assert.Equal(t, targetItems, newTargetItems)

}

// Tests that the newly added collector instance does not get assigned any target when the targets remain the same.
func TestNoAssignmentToNewCollector(t *testing.T) {
	s, _ := New("least-weighted", logger)

	// instantiate only 1 collector
	cols := MakeNCollectors(1, 0)
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	assert.Len(t, s.Collectors(), expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.Collectors()[i.Name])
	}

	initialColsBeforeAddingNewCol := s.Collectors()
	initTargets := MakeNNewTargets(6, 0, 0)

	// test that targets and collectors are added properly
	s.SetTargets(initTargets)

	// verify
	expectedTargetLen := len(initTargets)
	targetItems := s.TargetItems()
	assert.Len(t, targetItems, expectedTargetLen)

	// add another collector
	newColName := fmt.Sprintf("collector-%d", len(cols))
	cols[newColName] = &Collector{
		Name:       newColName,
		NumTargets: 0,
	}
	s.SetCollectors(cols)

	// targets shall not change
	newTargetItems := s.TargetItems()
	assert.Equal(t, targetItems, newTargetItems)

	// initial collectors still should have the same targets
	for colName, col := range s.Collectors() {
		if colName != newColName {
			assert.Equal(t, initialColsBeforeAddingNewCol[colName], col)
		}
	}

	// new collector should have no targets
	newCollector := s.Collectors()[newColName]
	assert.Equal(t, 0, newCollector.NumTargets)
}

// Tests that the delta in number of targets per collector is less than 15% of an even distribution.
func TestCollectorBalanceWhenAddingAndRemovingAtRandom(t *testing.T) {

	// prepare allocator with 3 collectors and 'random' amount of targets
	s, _ := New("least-weighted", logger)

	cols := MakeNCollectors(3, 0)
	s.SetCollectors(cols)

	targets := MakeNNewTargets(27, 3, 0)
	s.SetTargets(targets)

	// Divisor needed to get 15%
	divisor := 6.7

	targetItemLen := len(s.TargetItems())
	collectors := s.Collectors()
	count := targetItemLen / len(collectors)
	percent := float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
		assert.InDelta(t, i.NumTargets, count, percent)
	}

	// removing targets at 'random'
	// Remove half of targets randomly
	toDelete := len(targets) / 2
	counter := 0
	for index := range targets {
		shouldDelete := rand.Intn(toDelete) //nolint:gosec
		if counter < shouldDelete {
			delete(targets, index)
		}
		counter++
	}
	s.SetTargets(targets)

	targetItemLen = len(s.TargetItems())
	collectors = s.Collectors()
	count = targetItemLen / len(collectors)
	percent = float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
	// adding targets at 'random'
	for _, item := range MakeNNewTargets(13, 3, 100) {
		targets[item.Hash()] = item
	}
	s.SetTargets(targets)

	targetItemLen = len(s.TargetItems())
	collectors = s.Collectors()
	count = targetItemLen / len(collectors)
	percent = float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
}

func TestTargetsWithNoCollectorsLeastWeighted(t *testing.T) {
	s, _ := New("least-weighted", logger)

	// Adding 10 new targets
	numItems := 10
	initTargets := MakeNNewTargetsWithEmptyCollectors(numItems, 0)
	s.SetTargets(initTargets)
	actualTargetItems := s.TargetItems()
	assert.Len(t, actualTargetItems, numItems)

	// Adding 5 new targets, and removing the old 10 targets
	numItemsUpdate := 5
	newTargets := MakeNNewTargetsWithEmptyCollectors(numItemsUpdate, 10)
	s.SetTargets(newTargets)
	actualTargetItems = s.TargetItems()
	assert.Len(t, actualTargetItems, numItemsUpdate)

	// Adding 5 new targets, and one existing target
	numItemsUpdate = 6
	newTargets = MakeNNewTargetsWithEmptyCollectors(numItemsUpdate, 14)
	s.SetTargets(newTargets)
	actualTargetItems = s.TargetItems()
	assert.Len(t, actualTargetItems, numItemsUpdate)

	// Adding collectors to test allocation
	numCols := 2
	cols := MakeNCollectors(2, 0)
	s.SetCollectors(cols)

	// Checking to see that there is no change to number of targets
	actualTargetItems = s.TargetItems()
	assert.Len(t, actualTargetItems, numItemsUpdate)
	// Checking to see collectors are added correctly
	actualCollectors := s.Collectors()
	assert.Len(t, actualCollectors, numCols)

	// Divisor needed to get 15%
	divisor := 6.7
	targetItemLen := len(actualTargetItems)
	count := targetItemLen / len(actualCollectors)
	percent := float64(targetItemLen) / divisor

	// Check to see targets are allocated with the expected delta
	for _, i := range actualCollectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
}
