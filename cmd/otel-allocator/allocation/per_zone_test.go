// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var loggerPerZone = logf.Log.WithName("unit-tests")

func TestAllocationPerZone(t *testing.T) {
	kubeClient := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-0",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-0",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-1",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-2",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-3",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-1",
				},
			},
		})

	perZoneStrategy := newPerZoneStrategy()
	perZoneAllocator := newAllocator(loggerPerZone, perZoneStrategy, WithKubeClient(kubeClient))

	cols := MakeNCollectors(3, 0)
	perZoneAllocator.SetCollectors(cols)

	firstTargetLabels := labels.Labels{
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
	}
	secondTargetLabels := labels.Labels{
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-1"},
	}
	thirdTargetLabels := labels.Labels{
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-2"},
	}
	forthTargetLabels := labels.Labels{
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-3"},
	}

	firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstTargetLabels, "")
	secondTarget := target.NewItem("sample-name", "0.0.0.1:8000", secondTargetLabels, "")
	thirdTarget := target.NewItem("sample-name", "0.0.0.2:8000", thirdTargetLabels, "")
	fourthTarget := target.NewItem("sample-name", "0.0.0.3:8000", forthTargetLabels, "")

	targetList := map[string]*target.Item{
		firstTarget.Hash():  firstTarget,
		secondTarget.Hash(): secondTarget,
		thirdTarget.Hash():  thirdTarget,
		fourthTarget.Hash(): fourthTarget,
	}

	perZoneAllocator.SetTargets(targetList)

	actualItems := perZoneAllocator.TargetItems()
	expectedTargetLen := len(targetList)
	assert.Len(t, actualItems, expectedTargetLen)

	// verify allocation to nodes
	for targetHash, item := range targetList {
		actualItem, found := actualItems[targetHash]
		assert.True(t, found, "target with hash %s not found", item.Hash())

		itemsForCollector := perZoneAllocator.GetTargetsForCollectorAndJob(actualItem.CollectorName, actualItem.JobName)

		if targetHash == firstTarget.Hash() {
			assert.Len(t, itemsForCollector, 1)
			assert.Equal(t, actualItem.CollectorName, "collector-0")
			continue
		}
		if targetHash == secondTarget.Hash() {
			assert.Len(t, itemsForCollector, 2)
			assert.Equal(t, actualItem.CollectorName, "collector-1")
			continue
		}
		if targetHash == thirdTarget.Hash() {
			assert.Len(t, itemsForCollector, 1)
			assert.Equal(t, actualItem.CollectorName, "collector-2")
			continue
		}
		if targetHash == fourthTarget.Hash() {
			assert.Len(t, itemsForCollector, 2)
			assert.Equal(t, actualItem.CollectorName, "collector-1")
			continue
		}
	}
}

// Test with no collector in a specific zone.
func TestTargetsWithZoneDoesNotHaveCollectors(t *testing.T) {
	kubeClient := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-0",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-0",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-1",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-2",
				},
			},
		})

	perZoneStrategy := newPerZoneStrategy()
	perZoneAllocator := newAllocator(loggerPerZone, perZoneStrategy, WithKubeClient(kubeClient))

	cols := MakeNCollectors(2, 0)
	perZoneAllocator.SetCollectors(cols)

	firstTargetLabels := labels.Labels{
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
	}
	secondTargetLabels := labels.Labels{
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-1"},
	}
	thirdTargetLabels := labels.Labels{ // won't have a collector in the zone-2
		{Name: "__meta_kubernetes_pod_node_name", Value: "node-2"},
	}

	firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstTargetLabels, "")
	secondTarget := target.NewItem("sample-name", "0.0.0.1:8000", secondTargetLabels, "")
	thirdTarget := target.NewItem("sample-name", "0.0.0.2:8000", thirdTargetLabels, "")

	targetList := map[string]*target.Item{
		firstTarget.Hash():  firstTarget,
		secondTarget.Hash(): secondTarget,
		thirdTarget.Hash():  thirdTarget,
	}

	perZoneAllocator.SetTargets(targetList)

	mapString := fmt.Sprintf("%v", perZoneAllocator.TargetItems())
	fmt.Println(mapString)

	actualItems := perZoneAllocator.TargetItems()
	expectedTargetLen := len(targetList)
	assert.Len(t, actualItems, expectedTargetLen)

	// verify allocation to nodes
	for targetHash, item := range targetList {
		actualItem, found := actualItems[targetHash]
		assert.True(t, found, "target with hash %s not found", item.Hash())

		itemsForCollector := perZoneAllocator.GetTargetsForCollectorAndJob(actualItem.CollectorName, actualItem.JobName)

		if targetHash == firstTarget.Hash() {
			assert.Len(t, itemsForCollector, 1)
			assert.Equal(t, actualItem.CollectorName, "collector-0")
			continue
		}
		if targetHash == secondTarget.Hash() {
			assert.Len(t, itemsForCollector, 1)
			assert.Equal(t, actualItem.CollectorName, "collector-1")
			continue
		}
		if targetHash == thirdTarget.Hash() {
			assert.Len(t, itemsForCollector, 0)
			assert.Equal(t, actualItem.CollectorName, "")
			continue
		}
	}
}
