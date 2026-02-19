// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
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
			targets = slices.Delete(targets, index, index)
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
	targets = append(targets, MakeNNewTargets(13, 3, 100)...)
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

// TestLeastWeightedJobDistribution verifies that when multiple jobs have targets,
// each job's targets are evenly distributed across collectors.
// This tests the job-name tiebreaker: when collectors have equal total targets,
// the one with fewer targets from the current job should be chosen.
func TestLeastWeightedJobDistribution(t *testing.T) {
	s, _ := New("least-weighted", logger)

	numCols := 5
	numTargetsPerJob := 100 // Use larger numbers for more stable distribution
	jobs := []string{"job-a", "job-b", "job-c"}

	cols := MakeNCollectors(numCols, 0)
	s.SetCollectors(cols)

	// Create and assign targets for each job
	var allTargets []*target.Item
	for i, job := range jobs {
		targets := MakeNTargetsForJob(numTargetsPerJob, job, i*numTargetsPerJob)
		allTargets = append(allTargets, targets...)
	}
	s.SetTargets(allTargets)

	expectedPerJobPerCollector := float64(numTargetsPerJob) / float64(numCols) // 100/5 = 20

	// For each job, count targets per collector
	for _, job := range jobs {
		perCollector := map[string]int{}
		for _, col := range s.Collectors() {
			targets := s.GetTargetsForCollectorAndJob(col.Name, job)
			perCollector[col.Name] = len(targets)
		}

		t.Logf("job %s distribution: %v (expected ~%.0f each)", job, perCollector, expectedPerJobPerCollector)

		// Each collector should have close to expectedPerJobPerCollector targets from this job
		// Allow 10% variance to account for target processing order
		for colName, count := range perCollector {
			assert.InDelta(t, expectedPerJobPerCollector, count, expectedPerJobPerCollector*0.1,
				"collector %s should have ~%.0f targets from job %s, got %d",
				colName, expectedPerJobPerCollector, job, count)
		}
	}
}

// TestWeightedLoadBalancing verifies that heavy and light targets are distributed
// so that WeightedLoad is balanced across collectors rather than just target count.
func TestWeightedLoadBalancing(t *testing.T) {
	s, _ := New("least-weighted", logger)

	numCols := 3
	cols := MakeNCollectors(numCols, 0)
	s.SetCollectors(cols)

	// Create 3 heavy targets (weight=10 each) and 30 light targets (weight=1 each)
	// Total weight = 3*10 + 30*1 = 60, expect ~20 per collector
	heavyTargets := MakeNTargetsWithWeightClass(3, "heavy-job", 0, "__target_allocation_weight", "heavy")
	lightTargets := MakeNTargetsWithWeightClass(30, "light-job", 100, "__target_allocation_weight", "light")

	allTargets := append(heavyTargets, lightTargets...)
	s.SetTargets(allTargets)

	collectors := s.Collectors()

	// Verify heavy targets are spread across collectors (not all on one)
	heavyPerCollector := map[string]int{}
	for _, col := range collectors {
		targets := s.GetTargetsForCollectorAndJob(col.Name, "heavy-job")
		heavyPerCollector[col.Name] = len(targets)
	}
	t.Logf("Heavy targets per collector: %v", heavyPerCollector)
	// Each collector should have at most 1 heavy target (3 heavy / 3 collectors)
	for colName, count := range heavyPerCollector {
		assert.LessOrEqual(t, count, 2, "collector %s should have at most 2 heavy targets, got %d", colName, count)
	}

	// Verify WeightedLoad is balanced across collectors
	var loads []int
	for _, col := range collectors {
		loads = append(loads, col.WeightedLoad)
	}
	t.Logf("WeightedLoad per collector: %v", loads)

	// Expected total weight = 60, expected per collector = 20
	expectedPerCollector := 60.0 / float64(numCols)
	for _, col := range collectors {
		assert.InDelta(t, expectedPerCollector, col.WeightedLoad, expectedPerCollector*0.5,
			"collector %s WeightedLoad should be ~%.0f, got %d", col.Name, expectedPerCollector, col.WeightedLoad)
	}
}

// TestWeightedLoadUnlabeledTargets verifies that targets without a weight class label
// get the default weight of 1, so WeightedLoad equals NumTargets.
func TestWeightedLoadUnlabeledTargets(t *testing.T) {
	s, _ := New("least-weighted", logger)

	cols := MakeNCollectors(3, 0)
	s.SetCollectors(cols)

	targets := MakeNNewTargets(9, 3, 0)
	s.SetTargets(targets)

	for _, col := range s.Collectors() {
		assert.Equal(t, col.NumTargets, col.WeightedLoad,
			"unlabeled targets should have weight 1, so WeightedLoad equals NumTargets for collector %s", col.Name)
	}
}

// TestWeightedLoadUnknownClass verifies that targets with an unknown weight class
// use the default weight (light=1).
func TestWeightedLoadUnknownClass(t *testing.T) {
	s, _ := New("least-weighted", logger)

	cols := MakeNCollectors(1, 0)
	s.SetCollectors(cols)

	// Create targets with unknown weight class — should default to light (1)
	targets := MakeNTargetsWithWeightClass(2, "test-job", 0, "__target_allocation_weight", "unknown")
	s.SetTargets(targets)

	collectors := s.Collectors()
	for _, col := range collectors {
		// 2 targets * default weight 1 = 2
		assert.Equal(t, 2, col.WeightedLoad,
			"unknown weight class should use default weight (light=1)")
		assert.Equal(t, 2, col.NumTargets)
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
