package allocation

import (
	"math"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

// Tests the least connection - The expected collector after running findNextCollector should be the collector with the least amount of workload
func TestFindNextCollector(t *testing.T) {
	s := NewAllocator(logger)

	defaultCol := collector{Name: "default-col", NumTargets: 1}
	maxCol := collector{Name: "max-col", NumTargets: 2}
	leastCol := collector{Name: "least-col", NumTargets: 0}
	s.collectors[maxCol.Name] = &maxCol
	s.collectors[leastCol.Name] = &leastCol
	s.collectors[defaultCol.Name] = &defaultCol

	assert.Equal(t, "least-col", s.findNextCollector().Name)
}

func TestSetCollectors(t *testing.T) {
	s := NewAllocator(logger)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	collectors := s.Collectors()
	assert.Len(t, collectors, expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, collectors[i])
	}
}

func TestAddingAndRemovingTargets(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s := NewAllocator(logger)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)
	labels := model.LabelSet{}

	initTargets := []string{"prometheus:1000", "prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1005"}
	var targetList []TargetItem
	for _, i := range initTargets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: labels})
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify
	expectedTargetLen := len(initTargets)
	assert.Len(t, s.TargetItems(), expectedTargetLen)

	// prepare second round of targets
	tar := []string{"prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004"}
	var newTargetList []TargetItem
	for _, i := range tar {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: labels})
	}

	// test that fewer targets are found - removed
	s.SetTargets(newTargetList)

	// verify
	targetItems := s.TargetItems()
	expectedNewTargetLen := len(tar)
	assert.Len(t, targetItems, expectedNewTargetLen)

	// verify results map
	for _, i := range tar {
		_, ok := targetItems["sample-name"+i+labels.Fingerprint().String()]
		assert.True(t, ok)
	}
}

// Tests that two targets with the same target url and job name but different label set are both added
func TestAllocationCollision(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s := NewAllocator(logger)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)
	firstLabels := model.LabelSet{
		"test": "test1",
	}
	secondLabels := model.LabelSet{
		"test": "test2",
	}

	targetList := []TargetItem{
		{JobName: "sample-name", TargetURL: "0.0.0.0:8000", Label: firstLabels},
		{JobName: "sample-name", TargetURL: "0.0.0.0:8000", Label: secondLabels},
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify
	targetItems := s.TargetItems()
	expectedTargetLen := len(targetList)
	assert.Len(t, targetItems, expectedTargetLen)

	// verify results map
	for _, i := range targetList {
		_, ok := targetItems[i.hash()]
		assert.True(t, ok)
	}
}

func TestNoCollectorReassignment(t *testing.T) {
	s := NewAllocator(logger)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)
	labels := model.LabelSet{}

	expectedColLen := len(cols)
	assert.Len(t, s.collectors, expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.collectors[i])
	}
	initTargets := []string{"prometheus:1000", "prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1005"}
	var targetList []TargetItem
	for _, i := range initTargets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: labels})
	}
	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify
	expectedTargetLen := len(initTargets)
	targetItems := s.TargetItems()
	assert.Len(t, targetItems, expectedTargetLen)

	// assign new set of collectors with the same names
	newCols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(newCols)

	newTargetItems := s.TargetItems()
	assert.Equal(t, targetItems, newTargetItems)

}

func TestSmartCollectorReassignment(t *testing.T) {
	s := NewAllocator(logger)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)
	labels := model.LabelSet{}

	expectedColLen := len(cols)
	assert.Len(t, s.collectors, expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.collectors[i])
	}
	initTargets := []string{"prometheus:1000", "prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1005"}
	var targetList []TargetItem
	for _, i := range initTargets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: labels})
	}
	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify
	expectedTargetLen := len(initTargets)
	targetItems := s.TargetItems()
	assert.Len(t, targetItems, expectedTargetLen)

	// assign new set of collectors with the same names
	newCols := []string{"col-1", "col-2", "col-4"}
	s.SetCollectors(newCols)

	newTargetItems := s.TargetItems()
	assert.Equal(t, len(targetItems), len(newTargetItems))
	for key, targetItem := range targetItems {
		item, ok := newTargetItems[key]
		assert.True(t, ok, "all target items should be found in new target item list")
		if targetItem.Collector.Name != "col-3" {
			assert.Equal(t, targetItem.Collector.Name, item.Collector.Name)
		} else {
			assert.Equal(t, "col-4", item.Collector.Name)
		}
	}
}

// Tests that the delta in number of targets per collector is less than 15% of an even distribution
func TestCollectorBalanceWhenAddingAndRemovingAtRandom(t *testing.T) {

	// prepare allocator with 3 collectors and 'random' amount of targets
	s := NewAllocator(logger)

	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	targets := []string{"prometheus:1001", "prometheus:1002", "prometheus:1003", "prometheus:1004", "prometheus:1005", "prometheus:1006",
		"prometheus:1011", "prometheus:1012", "prometheus:1013", "prometheus:1014", "prometheus:1015", "prometheus:1016",
		"prometheus:1021", "prometheus:1022", "prometheus:1023", "prometheus:1024", "prometheus:1025", "prometheus:1026"}
	var newTargetList []TargetItem
	for _, i := range targets {
		newTargetList = append(newTargetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetTargets(newTargetList)

	// Divisor needed to get 15%
	divisor := 6.7

	targetItemLen := len(s.TargetItems())
	collectors := s.Collectors()
	count := targetItemLen / len(collectors)
	percent := float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
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
	s.SetTargets(newTargetList)

	targetItemLen = len(s.TargetItems())
	collectors = s.Collectors()
	count = targetItemLen / len(collectors)
	percent = float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
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
	s.SetTargets(newTargetList)

	targetItemLen = len(s.TargetItems())
	collectors = s.Collectors()
	count = targetItemLen / len(collectors)
	percent = float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
}
