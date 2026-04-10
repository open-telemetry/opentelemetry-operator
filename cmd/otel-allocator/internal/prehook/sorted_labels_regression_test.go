package prehook

import (
	"maps"
	"slices"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// TestSortedLabelsBlackboxRelabeling is a regression test for a bug where
// processTargetGroups produced unsorted Labels when group labels (e.g. vendor=nginx)
// sorted alphabetically after target labels (e.g. __address__). Labels.Get() uses
// binary search and silently returns empty on unsorted data, causing relabel rules
// that read source_labels: [__address__] to produce empty results. Both targets ended
// up with identical post-relabel labels, creating a hash collision that silently
// dropped one target in the allocator's buildTargetMap.
//
// The fix in processTargetGroups creates a fresh ScratchBuilder per target and calls
// Sort() after all labels are added, ensuring Labels.Get() works correctly.
func TestSortedLabelsBlackboxRelabeling(t *testing.T) {
	model.NameValidationScheme = model.UTF8Validation

	jobName := "blackbox-repro-with-label"

	// Build items using the FIXED processTargetGroups logic (globally sorted labels)
	items := buildItemsSortedLabels(jobName,
		map[model.LabelName]model.LabelValue{"vendor": "nginx"},
		[]string{
			"https://target-alpha.example.com:8393/",
			"https://target-beta.example.com:8393/",
		})

	// Verify labels are sorted (the fix's guarantee)
	for _, item := range items {
		var prevName string
		item.Labels.Range(func(l labels.Label) {
			if prevName != "" {
				assert.Less(t, prevName, l.Name, "Labels must be sorted")
			}
			prevName = l.Name
		})
	}

	// Apply prehook relabeling with standard blackbox-exporter pattern
	relabelCfgs := blackboxRelabelConfigs()
	prehook := New("relabel-config", logf.Log.WithName("regression-test"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: relabelCfgs})

	result := prehook.Apply(items)
	require.Len(t, result, 2, "Both targets must survive relabeling — collision drops one")

	// Verify unique hashes
	hashes := make(map[target.ItemHash]bool)
	for _, item := range result {
		hashes[item.Hash()] = true
	}
	assert.Len(t, hashes, 2, "Each target must have a unique hash after relabeling")
}

// TestNoGroupLabelAlwaysWorks verifies the no-label case works (was never broken).
func TestNoGroupLabelAlwaysWorks(t *testing.T) {
	model.NameValidationScheme = model.UTF8Validation

	jobName := "blackbox-no-label"
	items := buildItemsSortedLabels(jobName,
		map[model.LabelName]model.LabelValue{},
		[]string{
			"https://target-alpha.example.com:8393/",
			"https://target-beta.example.com:8393/",
		})

	prehook := New("relabel-config", logf.Log.WithName("regression-test"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: blackboxRelabelConfigs()})

	result := prehook.Apply(items)
	require.Len(t, result, 2, "Both targets must survive without group labels")
}

// blackboxRelabelConfigs returns the standard blackbox-exporter relabel pattern.
func blackboxRelabelConfigs() []*relabel.Config {
	return []*relabel.Config{
		{
			SourceLabels:         model.LabelNames{"__address__"},
			Separator:            ";",
			Regex:                relabel.MustNewRegexp("(.*)"),
			TargetLabel:          "__param_target",
			Replacement:          "$1",
			Action:               relabel.Replace,
			NameValidationScheme: model.UTF8Validation,
		},
		{
			SourceLabels:         model.LabelNames{"__param_target"},
			Separator:            ";",
			Regex:                relabel.MustNewRegexp("(.*)"),
			TargetLabel:          "instance",
			Replacement:          "$1",
			Action:               relabel.Replace,
			NameValidationScheme: model.UTF8Validation,
		},
		{
			SourceLabels:         model.LabelNames{},
			Separator:            ";",
			Regex:                relabel.MustNewRegexp("(.*)"),
			TargetLabel:          "__address__",
			Replacement:          "fake-blackbox.blackbox-repro.svc.cluster.local:9115",
			Action:               relabel.Replace,
			NameValidationScheme: model.UTF8Validation,
		},
	}
}

// buildItemsSortedLabels replicates the fixed processTargetGroups logic:
// fresh ScratchBuilder per target with Sort() to ensure globally sorted Labels.
func buildItemsSortedLabels(jobName string, groupLabelsMap map[model.LabelName]model.LabelValue, addresses []string) []*target.Item {
	const preallocSize = 16
	groupBuilder := labels.NewScratchBuilder(preallocSize)
	targetLabelNames := make([]string, 0, preallocSize)

	for ln, lv := range groupLabelsMap {
		groupBuilder.Add(string(ln), string(lv))
	}
	groupBuilder.Sort()
	groupLabels := groupBuilder.Labels()

	var items []*target.Item
	for _, addr := range addresses {
		tb := labels.NewScratchBuilder(preallocSize)
		groupLabels.Range(func(l labels.Label) {
			tb.Add(l.Name, l.Value)
		})
		targetLabelNames = targetLabelNames[:0]
		tgt := map[model.LabelName]model.LabelValue{model.AddressLabel: model.LabelValue(addr)}
		for ln := range maps.Keys(tgt) {
			targetLabelNames = append(targetLabelNames, string(ln))
		}
		slices.Sort(targetLabelNames)
		for _, ln := range targetLabelNames {
			tb.Add(ln, string(tgt[model.LabelName(ln)]))
		}
		tb.Sort()
		items = append(items, target.NewItem(jobName, addr, tb.Labels(), ""))
	}
	return items
}
