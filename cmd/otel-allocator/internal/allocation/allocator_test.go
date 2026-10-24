// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"context"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/featuregate"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

func TestSetCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		expectedColLen := len(cols)
		collectors := allocator.Collectors()
		assert.Len(t, collectors, expectedColLen)

		for _, i := range cols {
			assert.NotNil(t, collectors[i.Name])
		}
	})
}

func TestSetTargets(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		targets := MakeNNewTargetsWithEmptyCollectors(3, 0)
		allocator.SetTargets(targets)

		expectedTargetLen := len(targets)
		actualTargets := allocator.TargetItems()
		assert.Len(t, actualTargets, expectedTargetLen)
	})
}

func TestCanSetSingleTarget(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		targets := MakeNNewTargetsWithEmptyCollectors(1, 3)
		allocator.SetCollectors(cols)
		allocator.SetTargets(targets)
		actualTargetItems := allocator.TargetItems()
		assert.Len(t, actualTargetItems, 1)
		for _, item := range actualTargetItems {
			assert.NotEmpty(t, item.CollectorName)
		}
	})
}

func TestCanSetTargetsBeforeCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		targets := MakeNNewTargetsWithEmptyCollectors(1, 3)
		allocator.SetTargets(targets)
		allocator.SetCollectors(cols)
		actualTargetItems := allocator.TargetItems()
		assert.Len(t, actualTargetItems, 1)
		for _, item := range actualTargetItems {
			assert.NotEmpty(t, item.CollectorName)
		}
	})
}

func TestAddingAndRemovingTargets(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		initTargets := MakeNNewTargets(6, 3, 0)

		// test that targets and collectors are added properly
		allocator.SetTargets(initTargets)

		// verify
		expectedTargetLen := len(initTargets)
		assert.Len(t, allocator.TargetItems(), expectedTargetLen)

		// prepare second round of targets
		tar := MakeNNewTargets(4, 3, 0)

		// test that fewer targets are found - removed
		allocator.SetTargets(tar)

		// verify
		targetItems := allocator.TargetItems()
		expectedNewTargetLen := len(tar)
		assert.Len(t, targetItems, expectedNewTargetLen)

		// verify results map
		for _, i := range tar {
			_, ok := targetItems[i.Hash()]
			assert.True(t, ok)
		}
	})
}

func TestAddingAndRemovingCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		targets := MakeNNewTargetsWithEmptyCollectors(3, 0)
		allocator.SetTargets(targets)

		collectors := MakeNCollectors(3, 0)

		// test that targets and collectors are added properly
		allocator.SetCollectors(collectors)

		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		for _, tg := range allocator.TargetItems() {
			if tg.CollectorName != "" {
				assert.Contains(t, collectors, tg.CollectorName)
			}
		}

		// remove a collector
		delete(collectors, "collector-0")
		allocator.SetCollectors(collectors)
		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		for _, tg := range allocator.TargetItems() {
			if tg.CollectorName != "" {
				assert.Contains(t, collectors, tg.CollectorName)
			}
		}

		// add two more collectors
		collectors = MakeNCollectors(5, 0)
		allocator.SetCollectors(collectors)

		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		for _, tg := range allocator.TargetItems() {
			if tg.CollectorName != "" {
				assert.Contains(t, collectors, tg.CollectorName)
			}
		}

		// remove all collectors
		collectors = map[string]*Collector{}
		allocator.SetCollectors(collectors)

		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		jobs := []string{}
		for _, tg := range allocator.TargetItems() {
			assert.Empty(t, tg.CollectorName)
			jobs = append(jobs, tg.JobName)
		}
		for _, job := range jobs {
			for collector := range collectors {
				assert.Empty(t, allocator.GetTargetsForCollectorAndJob(collector, job))
			}
		}
	})
}

// Tests that two targets with the same target url and job name but different label set are both added.
func TestAllocationCollision(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)
		firstLabels := labels.New(labels.Label{Name: "test", Value: "test1"})
		secondLabels := labels.New(labels.Label{Name: "test", Value: "test2"})
		firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstLabels, "", target.HashLabels(firstLabels, "sample-name"))
		secondTarget := target.NewItem("sample-name", "0.0.0.0:8000", secondLabels, "", target.HashLabels(secondLabels, "sample-name"))

		targetList := []*target.Item{firstTarget, secondTarget}

		// test that targets and collectors are added properly
		allocator.SetTargets(targetList)

		// verify
		targetItems := allocator.TargetItems()
		expectedTargetLen := len(targetList)
		assert.Len(t, targetItems, expectedTargetLen)

		// verify results map
		for _, i := range targetList {
			_, ok := targetItems[i.Hash()]
			assert.True(t, ok)
		}
	})
}

func TestGetTargetsForCollectorAndJobNonExistent(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		targets := MakeNNewTargetsWithEmptyCollectors(6, 0)
		allocator.SetCollectors(cols)
		allocator.SetTargets(targets)

		// Non-existent collector returns empty slice
		result := allocator.GetTargetsForCollectorAndJob("non-existent-collector", "test-job-0")
		assert.Empty(t, result)

		// Non-existent job returns empty slice
		result = allocator.GetTargetsForCollectorAndJob("collector-0", "non-existent-job")
		assert.Empty(t, result)

		// Both non-existent returns empty slice
		result = allocator.GetTargetsForCollectorAndJob("no-collector", "no-job")
		assert.Empty(t, result)
	})
}

func TestSetEmptyTargets(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		// Set some targets first
		targets := MakeNNewTargetsWithEmptyCollectors(5, 0)
		allocator.SetTargets(targets)
		assert.Len(t, allocator.TargetItems(), 5)

		// Set empty targets - should clear all
		allocator.SetTargets([]*target.Item{})
		assert.Empty(t, allocator.TargetItems())

		// Collectors should still be present
		assert.Len(t, allocator.Collectors(), 3)
	})
}

func TestSetEmptyCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)
		targets := MakeNNewTargetsWithEmptyCollectors(6, 0)
		allocator.SetTargets(targets)

		// All targets should be assigned
		for _, item := range allocator.TargetItems() {
			assert.NotEmpty(t, item.CollectorName)
		}

		// Remove all collectors
		allocator.SetCollectors(map[string]*Collector{})
		assert.Empty(t, allocator.Collectors())

		// Targets should still exist but be unassigned
		assert.Len(t, allocator.TargetItems(), 6)
		for _, item := range allocator.TargetItems() {
			assert.Empty(t, item.CollectorName)
		}
	})
}

func TestTargetUpdatePreservesCount(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)
		targets := MakeNNewTargetsWithEmptyCollectors(10, 0)
		allocator.SetTargets(targets)

		// Update with same targets - counts should not change
		allocator.SetTargets(targets)

		totalAssigned := 0
		for _, col := range allocator.Collectors() {
			totalAssigned += col.NumTargets
		}
		assert.Equal(t, 10, totalAssigned)
	})
}

// TestSetTargetsUpdatesMetaLabelsForUnchangedHash covers
// https://github.com/open-telemetry/opentelemetry-operator/issues/4839: for a hostNetwork
// DaemonSet pod, __address__ (and therefore the target's hash, which excludes meta labels) stays
// the same across pod restarts, but meta labels such as __meta_kubernetes_pod_name change. The
// allocator must still surface the refreshed meta labels, while keeping the existing collector
// assignment since the target's identity is unchanged.
func TestSetTargetsUpdatesMetaLabelsForUnchangedHash(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		jobName := "sample-job"
		nonMetaLabels := labels.New(labels.Label{Name: "test", Value: "test1"})
		hash := target.HashLabels(nonMetaLabels, jobName)

		firstLabels := labels.New(
			labels.Label{Name: "test", Value: "test1"},
			labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
			labels.Label{Name: "__meta_kubernetes_pod_name", Value: "pod-1"},
		)
		firstTarget := target.NewItem(jobName, "0.0.0.0:8000", firstLabels, "", hash)
		allocator.SetTargets([]*target.Item{firstTarget})

		targetItems := allocator.TargetItems()
		assert.Len(t, targetItems, 1)
		originalCollector := targetItems[hash].CollectorName
		assert.NotEmpty(t, originalCollector)

		secondLabels := labels.New(
			labels.Label{Name: "test", Value: "test1"},
			labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "node-0"},
			labels.Label{Name: "__meta_kubernetes_pod_name", Value: "pod-2"},
		)
		secondTarget := target.NewItem(jobName, "0.0.0.0:8000", secondLabels, "", hash)
		allocator.SetTargets([]*target.Item{secondTarget})

		targetItems = allocator.TargetItems()
		assert.Len(t, targetItems, 1)
		assert.Equal(t, "pod-2", targetItems[hash].Labels.Get("__meta_kubernetes_pod_name"))
		assert.Equal(t, originalCollector, targetItems[hash].CollectorName)
	})
}

func TestNewAllocatorInvalidStrategy(t *testing.T) {
	_, err := New("invalid-strategy", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unregistered strategy")
}

func TestGetRegisteredAllocatorNames(t *testing.T) {
	names := GetRegisteredAllocatorNames()
	assert.GreaterOrEqual(t, len(names), 3)
	assert.Contains(t, names, "consistent-hashing")
	assert.Contains(t, names, "least-weighted")
	assert.Contains(t, names, "per-node")
}

func TestNewCollector(t *testing.T) {
	col := NewCollector("my-collector", "node-1")
	assert.Equal(t, "my-collector", col.Name)
	assert.Equal(t, "node-1", col.NodeName)
	assert.Equal(t, 0, col.NumTargets)
	assert.NotNil(t, col.TargetsPerJob)
	assert.Equal(t, "my-collector", col.Hash())
	assert.Equal(t, "my-collector", col.String())
}

func TestRepeatedSetCollectorsIdempotent(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		targets := MakeNNewTargetsWithEmptyCollectors(9, 0)

		allocator.SetCollectors(cols)
		allocator.SetTargets(targets)

		firstSnapshot := make(map[string]string)
		for hash, item := range allocator.TargetItems() {
			firstSnapshot[hash.String()] = item.CollectorName
		}

		// Set the same collectors again - assignments should not change
		allocator.SetCollectors(cols)

		for hash, item := range allocator.TargetItems() {
			assert.Equal(t, firstSnapshot[hash.String()], item.CollectorName,
				"target %s assignment changed after idempotent SetCollectors", hash)
		}
	})
}

func TestMultiJobAllocation(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		job1Targets := MakeNTargetsForJob(3, "job-alpha", 0)
		job2Targets := MakeNTargetsForJob(3, "job-beta", 100)
		allTargets := append(job1Targets, job2Targets...)

		allocator.SetTargets(allTargets)
		assert.Len(t, allocator.TargetItems(), 6)

		// All targets should be tracked (per-node may leave some unassigned
		// since MakeNTargetsForJob doesn't add node labels)
		assignedCount := 0
		for _, item := range allocator.TargetItems() {
			if item.CollectorName != "" {
				assignedCount++
			}
		}
		// For least-weighted and consistent-hashing, all should be assigned
		// For per-node, none will be assigned due to missing node labels
		assert.True(t, assignedCount == 0 || assignedCount == 6,
			"expected all targets assigned or none, got %d/6", assignedCount)
	})
}

func TestTargetsRemainingMetric(t *testing.T) {
	newTarget := func(job, namespace, address string) *target.Item {
		ls := labels.New(
			labels.Label{Name: "__address__", Value: address},
			labels.Label{Name: "__meta_kubernetes_namespace", Value: namespace},
		)
		return target.NewItem(job, address, ls, "", target.HashLabels(ls, job))
	}
	newAllocatorWithReader := func(t *testing.T) (Allocator, *sdkmetric.ManualReader) {
		reader := sdkmetric.NewManualReader()
		otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
		allocator, err := New(consistentHashingStrategyName, logf.Log.WithName("unit-tests"))
		require.NoError(t, err)
		return allocator, reader
	}
	collect := func(t *testing.T, reader *sdkmetric.ManualReader) metricdata.Gauge[int64] {
		var rm metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(context.Background(), &rm))
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "opentelemetry_allocator_targets_remaining" {
					return m.Data.(metricdata.Gauge[int64])
				}
			}
		}
		require.Fail(t, "opentelemetry_allocator_targets_remaining not recorded")
		return metricdata.Gauge[int64]{}
	}
	point := func(value int64, attrs ...attribute.KeyValue) metricdata.DataPoint[int64] {
		return metricdata.DataPoint[int64]{Attributes: attribute.NewSet(attrs...), Value: value}
	}

	t.Run("feature gate disabled", func(t *testing.T) {
		allocator, reader := newAllocatorWithReader(t)

		allocator.SetTargets([]*target.Item{
			newTarget("job-a", "ns-1", "10.0.0.1:8080"),
			newTarget("job-a", "ns-2", "10.0.0.2:8080"),
			newTarget("job-b", "", "10.0.0.3:8080"),
		})

		metricdatatest.AssertEqual(t, metricdata.Gauge[int64]{
			DataPoints: []metricdata.DataPoint[int64]{point(3)},
		}, collect(t, reader), metricdatatest.IgnoreTimestamp())
	})

	t.Run("feature gate enabled", func(t *testing.T) {
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.TargetsRemainingAttributes.ID(), true))
		t.Cleanup(func() {
			assert.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.TargetsRemainingAttributes.ID(), false))
		})
		allocator, reader := newAllocatorWithReader(t)

		allocator.SetTargets([]*target.Item{
			newTarget("job-a", "ns-1", "10.0.0.1:8080"),
			newTarget("job-a", "ns-1", "10.0.0.2:8080"),
			newTarget("job-a", "ns-2", "10.0.0.3:8080"),
			newTarget("job-b", "", "10.0.0.4:8080"),
		})

		metricdatatest.AssertEqual(t, metricdata.Gauge[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				point(2, attribute.String("job.name", "job-a"), attribute.String("k8s.namespace.name", "ns-1")),
				point(1, attribute.String("job.name", "job-a"), attribute.String("k8s.namespace.name", "ns-2")),
				point(1, attribute.String("job.name", "job-b")),
			},
		}, collect(t, reader), metricdatatest.IgnoreTimestamp())

		// Series for a job and namespace without targets drop to zero.
		allocator.SetTargets([]*target.Item{
			newTarget("job-a", "ns-1", "10.0.0.1:8080"),
		})

		metricdatatest.AssertEqual(t, metricdata.Gauge[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				point(1, attribute.String("job.name", "job-a"), attribute.String("k8s.namespace.name", "ns-1")),
				point(0, attribute.String("job.name", "job-a"), attribute.String("k8s.namespace.name", "ns-2")),
				point(0, attribute.String("job.name", "job-b")),
			},
		}, collect(t, reader), metricdatatest.IgnoreTimestamp())
	})
}
