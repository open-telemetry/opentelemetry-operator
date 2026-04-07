// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestItemHash_String(t *testing.T) {
	tests := []struct {
		name string
		h    ItemHash
		want string
	}{
		{
			name: "empty",
			h:    0,
			want: "0",
		},
		{
			name: "non-empty",
			h:    1,
			want: "1",
		},
		{
			name: "large value",
			h:    18446744073709551615,
			want: "18446744073709551615",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.h.String(), "String()")
		})
	}
}

func TestNewItem(t *testing.T) {
	ls := labels.New(labels.Label{Name: "app", Value: "test"})
	item := NewItem("job-1", "http://localhost:8080/metrics", ls, "collector-0")

	assert.Equal(t, "job-1", item.JobName)
	assert.Equal(t, "http://localhost:8080/metrics", item.TargetURL)
	assert.Equal(t, "collector-0", item.CollectorName)
	assert.Equal(t, ls, item.Labels)
}

func TestNewItemWithHash(t *testing.T) {
	ls := labels.New(labels.Label{Name: "app", Value: "test"})
	precomputedHash := ItemHash(42)
	item := NewItem("job-1", "http://localhost:8080", ls, "", WithHash(precomputedHash))

	assert.Equal(t, precomputedHash, item.Hash())
}

func TestItemHashStability(t *testing.T) {
	ls := labels.New(
		labels.Label{Name: "app", Value: "frontend"},
		labels.Label{Name: "env", Value: "prod"},
	)
	item1 := NewItem("my-job", "http://10.0.0.1:8080", ls, "")
	item2 := NewItem("my-job", "http://10.0.0.1:8080", ls, "")

	// Same inputs must produce the same hash
	assert.Equal(t, item1.Hash(), item2.Hash())

	// Hash must be non-zero for real items
	assert.NotZero(t, item1.Hash())
}

func TestItemHashDifferentJobs(t *testing.T) {
	ls := labels.New(labels.Label{Name: "app", Value: "test"})
	item1 := NewItem("job-a", "http://10.0.0.1:8080", ls, "")
	item2 := NewItem("job-b", "http://10.0.0.1:8080", ls, "")

	// Different job names should produce different hashes
	assert.NotEqual(t, item1.Hash(), item2.Hash())
}

func TestItemHashDifferentLabels(t *testing.T) {
	ls1 := labels.New(labels.Label{Name: "version", Value: "v1"})
	ls2 := labels.New(labels.Label{Name: "version", Value: "v2"})
	item1 := NewItem("job", "http://10.0.0.1:8080", ls1, "")
	item2 := NewItem("job", "http://10.0.0.1:8080", ls2, "")

	assert.NotEqual(t, item1.Hash(), item2.Hash())
}

func TestItemHashCollectorNameNotAffectHash(t *testing.T) {
	ls := labels.New(labels.Label{Name: "app", Value: "test"})
	item1 := NewItem("job", "http://10.0.0.1:8080", ls, "collector-0")
	item2 := NewItem("job", "http://10.0.0.1:8080", ls, "collector-1")

	// Collector name is not part of the hash
	assert.Equal(t, item1.Hash(), item2.Hash())
}

func TestGetNodeName(t *testing.T) {
	tests := []struct {
		name     string
		labels   labels.Labels
		expected string
	}{
		{
			name: "pod node name label",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "node-1"},
			),
			expected: "node-1",
		},
		{
			name: "kubernetes node name label",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_node_name", Value: "node-2"},
			),
			expected: "node-2",
		},
		{
			name: "endpoint node name label",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_endpoint_node_name", Value: "node-3"},
			),
			expected: "node-3",
		},
		{
			name: "endpointslice target kind Node",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_endpointslice_address_target_kind", Value: "Node"},
				labels.Label{Name: "__meta_kubernetes_endpointslice_address_target_name", Value: "node-4"},
			),
			expected: "node-4",
		},
		{
			name: "endpointslice target kind Pod (not Node)",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_endpointslice_address_target_kind", Value: "Pod"},
				labels.Label{Name: "__meta_kubernetes_endpointslice_address_target_name", Value: "pod-1"},
			),
			expected: "",
		},
		{
			name:     "no node labels",
			labels:   labels.New(labels.Label{Name: "app", Value: "test"}),
			expected: "",
		},
		{
			name:     "empty labels",
			labels:   labels.EmptyLabels(),
			expected: "",
		},
		{
			name: "pod node name takes priority",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "node-primary"},
				labels.Label{Name: "__meta_kubernetes_node_name", Value: "node-secondary"},
			),
			expected: "node-primary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := NewItem("job", "http://10.0.0.1:8080", tt.labels, "")
			assert.Equal(t, tt.expected, item.GetNodeName())
		})
	}
}

func TestGetEndpointSliceName(t *testing.T) {
	tests := []struct {
		name     string
		labels   labels.Labels
		expected string
	}{
		{
			name: "has endpointslice name",
			labels: labels.New(
				labels.Label{Name: "__meta_kubernetes_endpointslice_name", Value: "my-svc-abc12"},
			),
			expected: "my-svc-abc12",
		},
		{
			name:     "no endpointslice name",
			labels:   labels.New(labels.Label{Name: "app", Value: "test"}),
			expected: "",
		},
		{
			name:     "empty labels",
			labels:   labels.EmptyLabels(),
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := NewItem("job", "http://10.0.0.1:8080", tt.labels, "")
			assert.Equal(t, tt.expected, item.GetEndpointSliceName())
		})
	}
}

func TestLabelsHashWithJobName(t *testing.T) {
	ls := labels.New(
		labels.Label{Name: "app", Value: "test"},
		labels.Label{Name: "env", Value: "prod"},
	)

	hash1 := LabelsHashWithJobName(ls, "job-a")
	hash2 := LabelsHashWithJobName(ls, "job-a")
	hash3 := LabelsHashWithJobName(ls, "job-b")

	// Same inputs produce the same hash
	assert.Equal(t, hash1, hash2)
	// Different job names produce different hashes
	assert.NotEqual(t, hash1, hash3)
	// Hash is non-zero
	assert.NotZero(t, hash1)
}

func TestHashFromBuilder(t *testing.T) {
	ls := labels.New(
		labels.Label{Name: "app", Value: "test"},
		labels.Label{Name: "__meta_kubernetes_namespace", Value: "default"},
	)
	builder := labels.NewBuilder(ls)
	hash := HashFromBuilder(builder, "my-job")

	// Meta labels are skipped, so the hash should only consider "app=test"
	lsNoMeta := labels.New(labels.Label{Name: "app", Value: "test"})
	builderNoMeta := labels.NewBuilder(lsNoMeta)
	hashNoMeta := HashFromBuilder(builderNoMeta, "my-job")

	assert.Equal(t, hash, hashNoMeta, "meta labels should be skipped in hash computation")
	assert.NotZero(t, hash)
}

func TestHashFromBuilderDifferentJobs(t *testing.T) {
	ls := labels.New(labels.Label{Name: "app", Value: "test"})
	builder1 := labels.NewBuilder(ls)
	builder2 := labels.NewBuilder(ls)

	hash1 := HashFromBuilder(builder1, "job-1")
	hash2 := HashFromBuilder(builder2, "job-2")

	assert.NotEqual(t, hash1, hash2)
}

func TestItemHashCaching(t *testing.T) {
	ls := labels.New(labels.Label{Name: "app", Value: "test"})
	item := NewItem("job", "url", ls, "")

	// First call computes the hash
	h1 := item.Hash()
	// Second call should return the cached value
	h2 := item.Hash()

	assert.Equal(t, h1, h2)
	assert.NotZero(t, h1)
}
