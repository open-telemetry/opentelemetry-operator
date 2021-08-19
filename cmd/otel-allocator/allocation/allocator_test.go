package allocation

import (
	"fmt"
	"math"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

// Tests least connection - The expected collector after running findNextCollector should be the collector with the least amount of workload
func TestFindNextCollector(t *testing.T) {
	s := NewAllocator()
	defaultCol := collector{Name: "default-col", NumTargets: 1}
	maxCol := collector{Name: "max-col", NumTargets: 2}
	leastCol := collector{Name: "least-col", NumTargets: 0}
	s.collectors[maxCol.Name] = &maxCol
	s.collectors[leastCol.Name] = &leastCol
	s.collectors[defaultCol.Name] = &defaultCol

	assert.Equal(t, "least-col", s.findNextCollector().Name)
}

func TestSetCollectors(t *testing.T) {
	cols := []string{"col-1", "col-2", "col-3"}

	s := NewAllocator()
	s.SetCollectors(cols)

	excpectedColLen := len(cols)
	assert.Len(t, s.collectors, excpectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.collectors[i])
	}
}

func TestAddingAndRemovingTargets(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s := NewAllocator()
	cols := []string{"col-1", "col-2", "col-3"}
	initTargets := []string{"targ:1000", "targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005"}
	s.SetCollectors(cols)
	var targetList []TargetItem
	for _, i := range initTargets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}

	// test that targets and collectors are added properly
	s.SetWaitingTargets(targetList)
	s.AllocateTargets()

	// verify
	expectedTargetLen := len(initTargets)
	assert.Len(t, s.targetItems, expectedTargetLen)

	// prepare second round of targets
	tar := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004"}
	var newTargetList []TargetItem
	for _, i := range tar {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}

	// test that less targets are found - removed
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	// verify
	expectedNewTargetLen := len(tar)
	assert.Len(t, s.targetItems, expectedNewTargetLen)

	// verify results map
	for _, i := range tar {
		_, ok := s.targetItems["sample-name"+i]
		assert.True(t, ok)
	}
}

func TestCollectorBalanceWhenAddingTargets(t *testing.T) {

	// prepare allocator with four collectors and 100 targets
	s := NewAllocator()
	cols := []string{"col-1", "col-2", "col-3", "col-4"}
	var targetList []TargetItem
	targetCount := 100
	for i := 0; i < targetCount; i++ {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: fmt.Sprintf("targ:%d", i), Label: model.LabelSet{}})
	}

	// test the allocation
	s.SetCollectors(cols)
	s.SetWaitingTargets(targetList)
	s.AllocateTargets()

	// verify that each collector has the same amount of targets
	evenAmount := targetCount / len(cols)
	for _, i := range s.collectors {
		assert.Equal(t, evenAmount, i.NumTargets)
	}

}

// Tests that the delta in number of targets per collector is less than 15% of an even distribution
func TestCollectorBalanceWhenAddingAndRemovingAtRandom(t *testing.T) {

	// Divisor needed to get 15%
	divisor := 6.7

	// prepare allocator with 3 collectors and 'random' amount of targets
	s := NewAllocator()
	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	targets := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1015", "targ:1016",
		"targ:1021", "targ:1022", "targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	var newTargetList []TargetItem
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	count := len(s.targetItems) / len(s.collectors)
	percent := float64(len(s.targetItems)) / divisor

	// test
	for _, i := range s.collectors {
		assert.InDelta(t, i.NumTargets, count, percent)
	}

	// removing targets at 'random'
	targets = []string{"targ:1002", "targ:1003", "targ:1004", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1016",
		"targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	newTargetList = []TargetItem{}
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	count = len(s.targetItems) / len(s.collectors)
	percent = float64(len(s.targetItems)) / divisor

	// test
	for _, i := range s.collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
	// adding targets at 'random'
	targets = []string{"targ:1002", "targ:1003", "targ:1004", "targ:1006",
		"targ:1011", "targ:1012", "targ:1001", "targ:1014", "targ:1016",
		"targ:1023", "targ:1024", "targ:1025", "targ:1126", "targ:1227"}
	newTargetList = []TargetItem{}
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(newTargetList)
	s.AllocateTargets()

	count = len(s.targetItems) / len(s.collectors)
	percent = float64(len(s.targetItems)) / divisor

	// test
	for _, i := range s.collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
}

func TestCollectorBalanceWhenCollectorsChanged(t *testing.T) {
	s := NewAllocator()
	cols := []string{"col-1", "col-2"}
	s.SetCollectors(cols)

	targets := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1015", "targ:1016",
		"targ:1021", "targ:1022", "targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	var targetList []TargetItem
	for _, i := range targets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(targetList)
	s.AllocateTargets()

	// 18 / 2
	expected := 9
	for _, i := range s.collectors {
		assert.Equal(t, expected, i.NumTargets)
	}

	cols = []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)
	s.ReallocateCollectors()

	// 15 / 3
	expected = 6
	for _, i := range s.collectors {
		assert.Equal(t, expected, i.NumTargets)
	}
}
