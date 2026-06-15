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

func TestMergeSortedLabels(t *testing.T) {
	tests := []struct {
		name             string
		groupLabels      []labels.Label
		targetLabelNames []model.LabelName
		targetLabels     model.LabelSet
		expected         labels.Labels
	}{
		{
			name:             "empty inputs",
			groupLabels:      nil,
			targetLabelNames: nil,
			targetLabels:     nil,
			expected:         labels.EmptyLabels(),
		},
		{
			name:        "only group labels",
			groupLabels: []labels.Label{{Name: "env", Value: "prod"}, {Name: "region", Value: "us"}},
			expected:    labels.FromStrings("env", "prod", "region", "us"),
		},
		{
			name:             "only target labels",
			targetLabelNames: []model.LabelName{"__address__"},
			targetLabels:     model.LabelSet{"__address__": "localhost:9090"},
			expected:         labels.FromStrings("__address__", "localhost:9090"),
		},
		{
			name:             "interleaved merge",
			groupLabels:      []labels.Label{{Name: "env", Value: "prod"}},
			targetLabelNames: []model.LabelName{"__address__"},
			targetLabels:     model.LabelSet{"__address__": "localhost:9090"},
			expected:         labels.FromStrings("__address__", "localhost:9090", "env", "prod"),
		},
		{
			name:             "group label sorts after target label (original bug scenario)",
			groupLabels:      []labels.Label{{Name: "vendor", Value: "nginx"}},
			targetLabelNames: []model.LabelName{"__address__"},
			targetLabels:     model.LabelSet{"__address__": "https://target-alpha.example.com:8393/"},
			expected:         labels.FromStrings("__address__", "https://target-alpha.example.com:8393/", "vendor", "nginx"),
		},
		{
			name:             "target overrides group on collision",
			groupLabels:      []labels.Label{{Name: "job", Value: "from-group"}},
			targetLabelNames: []model.LabelName{"job"},
			targetLabels:     model.LabelSet{"job": "from-target"},
			expected:         labels.FromStrings("job", "from-target"),
		},
		{
			name:             "multiple labels both sides",
			groupLabels:      []labels.Label{{Name: "b", Value: "2"}, {Name: "d", Value: "4"}},
			targetLabelNames: []model.LabelName{"a", "c", "e"},
			targetLabels:     model.LabelSet{"a": "1", "c": "3", "e": "5"},
			expected:         labels.FromStrings("a", "1", "b", "2", "c", "3", "d", "4", "e", "5"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := labels.NewScratchBuilder(16)
			mergeSortedLabels(&builder, tt.groupLabels, tt.targetLabelNames, tt.targetLabels)
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
	for _, addr := range addresses {
		builder := labels.NewScratchBuilder(16)
		targetLabelNames := []model.LabelName{model.AddressLabel}
		targetLabels := model.LabelSet{model.AddressLabel: model.LabelValue(addr)}
		mergeSortedLabels(&builder, groupLabels, targetLabelNames, targetLabels)
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
