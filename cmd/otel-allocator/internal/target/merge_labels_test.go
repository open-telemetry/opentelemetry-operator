// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeLabels(t *testing.T) {
	tests := []struct {
		name         string
		groupLabels  []labels.Label
		targetLabels model.LabelSet
		expected     labels.Labels
	}{
		{
			name:     "empty inputs",
			expected: labels.EmptyLabels(),
		},
		{
			name:        "only group labels",
			groupLabels: []labels.Label{{Name: "env", Value: "prod"}, {Name: "region", Value: "us"}},
			expected:    labels.FromStrings("env", "prod", "region", "us"),
		},
		{
			name:         "only target labels",
			targetLabels: model.LabelSet{"__address__": "localhost:9090"},
			expected:     labels.FromStrings("__address__", "localhost:9090"),
		},
		{
			name:         "interleaved merge",
			groupLabels:  []labels.Label{{Name: "env", Value: "prod"}},
			targetLabels: model.LabelSet{"__address__": "localhost:9090"},
			expected:     labels.FromStrings("__address__", "localhost:9090", "env", "prod"),
		},
		{
			name:         "group label sorts after target label (original bug scenario)",
			groupLabels:  []labels.Label{{Name: "vendor", Value: "nginx"}},
			targetLabels: model.LabelSet{"__address__": "https://target-alpha.example.com:8393/"},
			expected:     labels.FromStrings("__address__", "https://target-alpha.example.com:8393/", "vendor", "nginx"),
		},
		{
			name:         "target overrides group on collision",
			groupLabels:  []labels.Label{{Name: "job", Value: "from-group"}},
			targetLabels: model.LabelSet{"job": "from-target"},
			expected:     labels.FromStrings("job", "from-target"),
		},
		{
			name:        "multiple labels both sides",
			groupLabels: []labels.Label{{Name: "env", Value: "prod"}, {Name: "namespace", Value: "monitoring"}},
			targetLabels: model.LabelSet{
				"__address__":      "localhost:9090",
				"__metrics_path__": "/metrics",
				"__scheme__":       "https",
			},
			expected: labels.FromStrings(
				"__address__", "localhost:9090",
				"__metrics_path__", "/metrics",
				"__scheme__", "https",
				"env", "prod",
				"namespace", "monitoring",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := labels.NewScratchBuilder(16)
			targetLabelNamesBuf := make([]model.LabelName, 0, len(tt.targetLabels))
			mergeLabels(&builder, tt.groupLabels, tt.targetLabels, targetLabelNamesBuf)
			result := builder.Labels()
			assert.Equal(t, tt.expected, result)

			// Verify output is sorted
			var prevName string
			result.Range(func(l labels.Label) {
				if prevName != "" {
					require.Less(t, prevName, l.Name, "Labels must be sorted")
				}
				prevName = l.Name
			})
		})
	}
}

// TestSortedLabelsBlackboxRelabeling verifies that when group labels sort
// alphabetically after target labels (e.g. vendor > __address__), the merged
// Labels are globally sorted. This ensures Labels.Get() (binary search) works
// correctly, preventing hash collisions that silently drop targets.
func TestSortedLabelsBlackboxScenario(t *testing.T) {
	groupLabels := []labels.Label{{Name: "vendor", Value: "nginx"}}
	addresses := []string{
		"https://target-alpha.example.com:8393/",
		"https://target-beta.example.com:8393/",
	}

	var items []*Item
	targetLabelNamesBuf := make([]model.LabelName, 0, 1)
	for _, addr := range addresses {
		builder := labels.NewScratchBuilder(16)
		targetLabels := model.LabelSet{model.AddressLabel: model.LabelValue(addr)}
		targetLabelNamesBuf = targetLabelNamesBuf[:0]
		mergeLabels(&builder, groupLabels, targetLabels, targetLabelNamesBuf)
		items = append(items, NewItem("blackbox-test", addr, builder.Labels(), ""))
	}

	// Verify labels are sorted
	for _, item := range items {
		var prevName string
		item.Labels.Range(func(l labels.Label) {
			if prevName != "" {
				assert.Less(t, prevName, l.Name, "Labels must be sorted")
			}
			prevName = l.Name
		})
	}

	// Verify Get("__address__") works on each item (would fail on unsorted labels)
	for i, item := range items {
		addr := item.Labels.Get("__address__")
		assert.Equal(t, addresses[i], addr, "Labels.Get must return the correct address")
	}

	// Verify unique hashes — identical hashes would mean target loss
	hashes := make(map[ItemHash]bool)
	for _, item := range items {
		hashes[item.Hash()] = true
	}
	assert.Len(t, hashes, 2, "Each target must have a unique hash")
}
