// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Note: These utilities are used by other packages, which is why they're defined in a non-test file.

package allocation

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

func colIndex(index, numCols int) int {
	if numCols == 0 {
		return -1
	}
	return index % numCols
}

func MakeNNewTargets(n int, numCollectors int, startingIndex int) map[string]*target.Item {
	toReturn := map[string]*target.Item{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		label := labels.Labels{
			{Name: "i", Value: strconv.Itoa(i)},
			{Name: "total", Value: strconv.Itoa(n + startingIndex)},
		}
		newTarget := target.NewItem(fmt.Sprintf("test-job-%d", i), fmt.Sprintf("test-url-%d", i), label, collector)
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func MakeNCollectors(n int, startingIndex int) map[string]*Collector {
	toReturn := map[string]*Collector{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", i)
		toReturn[collector] = &Collector{
			Name:       collector,
			NumTargets: 0,
			NodeName:   fmt.Sprintf("node-%d", i),
		}
	}
	return toReturn
}

func MakeNNewTargetsWithEmptyCollectors(n int, startingIndex int) map[string]*target.Item {
	toReturn := map[string]*target.Item{}
	for i := startingIndex; i < n+startingIndex; i++ {
		label := labels.Labels{
			{Name: "i", Value: strconv.Itoa(i)},
			{Name: "total", Value: strconv.Itoa(n + startingIndex)},
			{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
		}
		newTarget := target.NewItem(fmt.Sprintf("test-job-%d", i), fmt.Sprintf("test-url-%d", i), label, "")
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func RunForAllStrategies(t *testing.T, f func(t *testing.T, allocator Allocator)) {
	allocatorNames := GetRegisteredAllocatorNames()
	logger := logf.Log.WithName("unit-tests")
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
					corev1.LabelTopologyZone: "zone-3",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-4",
				Labels: map[string]string{
					corev1.LabelTopologyZone: "zone-4",
				},
			},
		},
	)
	for _, allocatorName := range allocatorNames {
		t.Run(allocatorName, func(t *testing.T) {
			allocator, err := New(allocatorName, logger, WithKubeClient(kubeClient))
			require.NoError(t, err)
			f(t, allocator)
		})
	}
}
