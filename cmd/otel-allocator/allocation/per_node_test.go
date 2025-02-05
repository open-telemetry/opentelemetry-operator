// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var loggerPerNode = logf.Log.WithName("unit-tests")

func GetTargetsWithNodeName(targets []*target.Item) (targetsWithNodeName []*target.Item) {
	for _, item := range targets {
		if item.GetNodeName() != "" {
			targetsWithNodeName = append(targetsWithNodeName, item)
		}
	}
	return targetsWithNodeName
}

// Tests that four targets, with one of them lacking node labels, are assigned except for the
// target that lacks node labels.
func TestAllocationPerNode(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s, _ := New("per-node", loggerPerNode)

	cols := MakeNCollectors(4, 0)
	s.SetCollectors(cols)
	firstLabels := labels.Labels{
		{Name: "test", Value: "test1"},
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
	}
	secondLabels := labels.Labels{
		{Name: "test", Value: "test2"},
		{Name: "__meta_kubernetes_node_name", Value: "node-1"},
	}
	// no label, should be skipped
	thirdLabels := labels.Labels{
		{Name: "test", Value: "test3"},
	}
	// endpointslice target kind and name
	fourthLabels := labels.Labels{
		{Name: "test", Value: "test4"},
		{Name: "__meta_kubernetes_endpointslice_address_target_kind", Value: "Node"},
		{Name: "__meta_kubernetes_endpointslice_address_target_name", Value: "node-3"},
	}

	firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstLabels, "")
	secondTarget := target.NewItem("sample-name", "0.0.0.0:8000", secondLabels, "")
	thirdTarget := target.NewItem("sample-name", "0.0.0.0:8000", thirdLabels, "")
	fourthTarget := target.NewItem("sample-name", "0.0.0.0:8000", fourthLabels, "")

	targetList := map[string]*target.Item{
		firstTarget.Hash():  firstTarget,
		secondTarget.Hash(): secondTarget,
		thirdTarget.Hash():  thirdTarget,
		fourthTarget.Hash(): fourthTarget,
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify length
	actualItems := s.TargetItems()

	// one target should be skipped
	expectedTargetLen := len(targetList)
	assert.Len(t, actualItems, expectedTargetLen)

	// verify allocation to nodes
	for targetHash, item := range targetList {
		actualItem, found := actualItems[targetHash]
		// if third target, should be skipped
		assert.True(t, found, "target with hash %s not found", item.Hash())

		// only the first two targets should be allocated
		itemsForCollector := s.GetTargetsForCollectorAndJob(actualItem.CollectorName, actualItem.JobName)

		// first two should be assigned one to each collector; if third target, should not be assigned
		if targetHash == thirdTarget.Hash() {
			assert.Len(t, itemsForCollector, 0)
			continue
		}
		assert.Len(t, itemsForCollector, 1)
		assert.Equal(t, actualItem, itemsForCollector[0])
	}
}

// Tests that four targets, with one of them missing node labels, are all assigned.
func TestAllocationPerNodeUsingFallback(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s, _ := New("per-node", loggerPerNode, WithFallbackStrategy(consistentHashingStrategyName))

	cols := MakeNCollectors(4, 0)
	s.SetCollectors(cols)
	firstLabels := labels.Labels{
		{Name: "test", Value: "test1"},
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
	}
	secondLabels := labels.Labels{
		{Name: "test", Value: "test2"},
		{Name: "__meta_kubernetes_node_name", Value: "node-1"},
	}
	// no label, should be allocated by the fallback strategy
	thirdLabels := labels.Labels{
		{Name: "test", Value: "test3"},
	}
	// endpointslice target kind and name
	fourthLabels := labels.Labels{
		{Name: "test", Value: "test4"},
		{Name: "__meta_kubernetes_endpointslice_address_target_kind", Value: "Node"},
		{Name: "__meta_kubernetes_endpointslice_address_target_name", Value: "node-3"},
	}

	firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstLabels, "")
	secondTarget := target.NewItem("sample-name", "0.0.0.0:8000", secondLabels, "")
	thirdTarget := target.NewItem("sample-name", "0.0.0.0:8000", thirdLabels, "")
	fourthTarget := target.NewItem("sample-name", "0.0.0.0:8000", fourthLabels, "")

	targetList := map[string]*target.Item{
		firstTarget.Hash():  firstTarget,
		secondTarget.Hash(): secondTarget,
		thirdTarget.Hash():  thirdTarget,
		fourthTarget.Hash(): fourthTarget,
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify length
	actualItems := s.TargetItems()

	// all targets should be allocated
	expectedTargetLen := len(targetList)
	assert.Len(t, actualItems, expectedTargetLen)

	// verify allocation to nodes
	for targetHash, item := range targetList {
		actualItem, found := actualItems[targetHash]

		assert.True(t, found, "target with hash %s not found", item.Hash())

		itemsForCollector := s.GetTargetsForCollectorAndJob(actualItem.CollectorName, actualItem.JobName)

		// first two should be assigned one to each collector; if third target, it should be assigned
		// according to the fallback strategy which may assign it to the otherwise empty collector or
		// one of the others, depending on the strategy and collector loop order
		if targetHash == thirdTarget.Hash() {
			assert.Empty(t, item.GetNodeName())
			assert.NotZero(t, len(itemsForCollector))
			continue
		}

		// Only check targets that have been assigned using the per-node (not fallback) strategy here
		assert.Len(t, GetTargetsWithNodeName(itemsForCollector), 1)
		assert.Equal(t, actualItem, GetTargetsWithNodeName(itemsForCollector)[0])
	}
}

func TestTargetsWithNoCollectorsPerNode(t *testing.T) {
	// prepare allocator with initial targets and collectors
	c, _ := New("per-node", loggerPerNode)

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
	// Based on lable all targets should be assigned to node-0
	for name, ac := range actualCollectors {
		if name == "collector-0" {
			assert.Equal(t, 6, ac.NumTargets)
		} else {
			assert.Equal(t, 0, ac.NumTargets)
		}
	}
}
