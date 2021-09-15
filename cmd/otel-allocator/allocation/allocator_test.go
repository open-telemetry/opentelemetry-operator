package allocation

import (
	"math"
	"testing"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

// Tests least connection - The expected collector after running findNextCollector should be the collector with the least amount of workload
func TestFindNextCollector(t *testing.T) {
	var log logr.Logger
	s := NewAllocator(log)

	defaultCol := collector{Name: "default-col", NumTargets: 1}
	maxCol := collector{Name: "max-col", NumTargets: 2}
	leastCol := collector{Name: "least-col", NumTargets: 0}
	s.collectors[maxCol.Name] = &maxCol
	s.collectors[leastCol.Name] = &leastCol
	s.collectors[defaultCol.Name] = &defaultCol

	assert.Equal(t, "least-col", s.findNextCollector().Name)
}

func TestSetCollectors(t *testing.T) {

	var log logr.Logger
	s := NewAllocator(log)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	excpectedColLen := len(cols)
	assert.Len(t, s.collectors, excpectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.collectors[i])
	}
}

func TestAddingAndRemovingTargets(t *testing.T) {
	// prepare allocator with initial targets and collectors
	var log logr.Logger
	s := NewAllocator(log)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	initTargets := []string{"prometheus:1000", "prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1005"}
	var targetList []TargetItem
	for _, i := range initTargets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}

	// test that targets and collectors are added properly
	s.SetWaitingTargets(targetList)
	s.AllocateTargets()

	// verify
	expectedTargetLen := len(initTargets)
	assert.Len(t, s.TargetItems, expectedTargetLen)

	// prepare second round of targets
	tar := []string{"prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004"}
	var newTargetList []TargetItem
	for _, i := range tar {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}

	// test that less targets are found - removed
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	// verify
	expectedNewTargetLen := len(tar)
	assert.Len(t, s.TargetItems, expectedNewTargetLen)

	// verify results map
	for _, i := range tar {
		_, ok := s.TargetItems["sample-name"+i]
		assert.True(t, ok)
	}
}

// Tests that the delta in number of targets per collector is less than 15% of an even distribution
func TestCollectorBalanceWhenAddingAndRemovingAtRandom(t *testing.T) {

	// prepare allocator with 3 collectors and 'random' amount of targets
	var log logr.Logger
	s := NewAllocator(log)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	targets := []string{"prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1005", "prometheus:1006",
		"prometheus:1011", "prometheus:1012", "prometheus:1013", "prometheus:1014", "prometheus:1015", "prometheus:1016",
		"prometheus:1021", "prometheus:1022", "prometheus:1023", "prometheus:1024", "prometheus:1025", "prometheus:1026"}
	var newTargetList []TargetItem
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	// Divisor needed to get 15%
	divisor := 6.7

	count := len(s.TargetItems) / len(s.collectors)
	percent := float64(len(s.TargetItems)) / divisor

	// test
	for _, i := range s.collectors {
		assert.InDelta(t, i.NumTargets, count, percent)
	}

	// removing targets at 'random'
	targets = []string{"prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1006",
		"prometheus:1011", "prometheus:1012", "prometheus:1013", "prometheus:1014", "prometheus:1016",
		"prometheus:1023", "prometheus:1024", "prometheus:1025", "prometheus:1026"}
	newTargetList = []TargetItem{}
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	count = len(s.TargetItems) / len(s.collectors)
	percent = float64(len(s.TargetItems)) / divisor

	// test
	for _, i := range s.collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
	// adding targets at 'random'
	targets = []string{"prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1006",
		"prometheus:1011", "prometheus:1012", "prometheus:1001", "prometheus:1014", "prometheus:1016",
		"prometheus:1023", "prometheus:1024", "prometheus:1025", "prometheus:1126", "prometheus:1227"}
	newTargetList = []TargetItem{}
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	count = len(s.TargetItems) / len(s.collectors)
	percent = float64(len(s.TargetItems)) / divisor

	// test
	for _, i := range s.collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
}
