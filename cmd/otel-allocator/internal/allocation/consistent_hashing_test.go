// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

func TestRelativelyEvenDistribution(t *testing.T) {
	numCols := 15
	numItems := 10000
	cols := MakeNCollectors(numCols, 0)
	var expectedPerCollector = float64(numItems / numCols)
	expectedDelta := (expectedPerCollector * 1.5) - expectedPerCollector
	c, _ := New("consistent-hashing", logger)
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
	c, _ := New("consistent-hashing", logger)
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
	c, _ := New("consistent-hashing", logger)
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

func TestSameJobDistributedAcrossCollectors(t *testing.T) {
	numCols := 5
	numTargetsPerJob := 100
	numJobs := 3
	cols := MakeNCollectors(numCols, 0)
	c, _ := New("consistent-hashing", logger)
	c.SetCollectors(cols)

	// Create targets where many share the same job name
	var targets []*target.Item
	for j := 0; j < numJobs; j++ {
		jobName := fmt.Sprintf("shared-job-%d", j)
		for i := 0; i < numTargetsPerJob; i++ {
			label := labels.New(
				labels.Label{Name: "i", Value: strconv.Itoa(i)},
				labels.Label{Name: "job_index", Value: strconv.Itoa(j)},
			)
			tg := target.NewItem(jobName, fmt.Sprintf("test-url-%d-%d", j, i), label, "")
			targets = append(targets, tg)
		}
	}
	c.SetTargets(targets)

	assert.Len(t, c.TargetItems(), numJobs*numTargetsPerJob)

	// For each job, verify targets are spread across multiple collectors
	for j := 0; j < numJobs; j++ {
		jobName := fmt.Sprintf("shared-job-%d", j)
		collectorsWithJob := map[string]int{}
		for _, col := range c.Collectors() {
			tgs := c.GetTargetsForCollectorAndJob(col.Name, jobName)
			if len(tgs) > 0 {
				collectorsWithJob[col.Name] = len(tgs)
			}
		}
		t.Logf("job %s distributed across %d/%d collectors: %v", jobName, len(collectorsWithJob), numCols, collectorsWithJob)
		assert.Greater(t, len(collectorsWithJob), 1, "job %s should be distributed across multiple collectors", jobName)
	}
}

func TestTargetsWithNoCollectorsConsistentHashing(t *testing.T) {

	c, _ := New("consistent-hashing", logger)

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
	// Checking to see that there is no change to number of targets
	actualTargetItems = c.TargetItems()
	assert.Len(t, actualTargetItems, numItemsUpdate)
	// Checking to see collectors are added correctly
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numCols)
	totalAssigned := 0
	for _, col := range actualCollectors {
		assert.Greater(t, col.NumTargets, 0, "each collector should have at least one target")
		totalAssigned += col.NumTargets
	}
	assert.Equal(t, numItemsUpdate, totalAssigned)
}
