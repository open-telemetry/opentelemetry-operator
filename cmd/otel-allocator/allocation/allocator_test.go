package allocation

import (
	"strconv"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

// Tests least connection - The expected collector after running findNextCollector should be the collecter with the least amount of workload
func TestFindNextCollector(t *testing.T) {
	s := NewAllocator()
	defaultCol := collector{Name: "default-col", NumTargets: 1}
	maxCol := collector{Name: "max-col", NumTargets: 2}
	leastCol := collector{Name: "least-col", NumTargets: 0}
	s.collectors[maxCol.Name] = &maxCol
	s.collectors[leastCol.Name] = &leastCol
	s.collectors[defaultCol.Name] = &defaultCol
	//s.nextCollector = &defaultCol

	//s.findNextCollector().Name
	assert.Equal(t, "least-col", s.findNextCollector().Name)
}

func TestSetCollectors(t *testing.T) {
	cols := []string{"col-1", "col-2", "col-3"}

	s := NewAllocator()
	s.SetCollectors(cols)

	assert.Equal(t, len(cols), len(s.collectors))
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
	s.Reallocate()

	// verify
	assert.True(t, len(s.targetItems) == 6)

	// prepare second round of targets
	tar := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004"}
	var tarL []TargetItem
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}

	// test that less targets are found - removed
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	// verify
	assert.True(t, len(s.targetItems) == 4)

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
	for i := 0; i < 100; i++ {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: "targ:10" + strconv.Itoa(i), Label: model.LabelSet{}})
	}

	// test the allocation
	s.SetCollectors(cols)
	s.SetWaitingTargets(targetList)
	s.Reallocate()

	// verify that each collector has the same amount of targets
	for _, i := range s.collectors {
		assert.True(t, i.NumTargets == 25)
	}

}

func TestCollectorBalanceWhenAddingAndRemovingAtRandom(t *testing.T) {

	// prepare allocator with 3 collectors and 'random' amount of targets
	s := NewAllocator()
	cols := []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)

	tar := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1015", "targ:1016",
		"targ:1021", "targ:1022", "targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	var tarL []TargetItem
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	// removing targets at 'random'
	tar = []string{"targ:1002", "targ:1003", "targ:1004", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1016",
		"targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	tarL = []TargetItem{}
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	// adding targets at 'random'
	tar = []string{"targ:1002", "targ:1003", "targ:1004", "targ:1006",
		"targ:1011", "targ:1012", "targ:1001", "targ:1014", "targ:1016",
		"targ:1023", "targ:1024", "targ:1025", "targ:1126", "targ:1227"}
	tarL = []TargetItem{}
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	count := len(s.targetItems) / len(s.collectors)
	percent := float64(len(s.targetItems)) / 6.7
	upperBound := float64(count) + percent
	lowerBound := float64(count) - percent

	for _, i := range s.collectors {
		assert.True(t, float64(i.NumTargets) >= lowerBound && float64(i.NumTargets) <= upperBound)
	}
}

func TestCollectorBalanceWhenCollectorsChanged(t *testing.T) {
	s := NewAllocator()
	cols := []string{"col-1", "col-2"}
	s.SetCollectors(cols)

	tar := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1015", "targ:1016",
		"targ:1021", "targ:1022", "targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	var tarL []TargetItem
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	// removing targets at 'random'
	tar = []string{"targ:1002", "targ:1003", "targ:1004", "targ:1006",
		"targ:1011", "targ:1012", "targ:1013", "targ:1014", "targ:1016",
		"targ:1023", "targ:1024", "targ:1025", "targ:1026"}
	tarL = []TargetItem{}
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	// adding targets at 'random'
	tar = []string{"targ:1002", "targ:1003", "targ:1004", "targ:1006",
		"targ:1011", "targ:1012", "targ:1001", "targ:1014", "targ:1016",
		"targ:1023", "targ:1024", "targ:1025", "targ:1126", "targ:1227", "targ:3030"}
	tarL = []TargetItem{}
	for _, i := range tar {
		tarL = append(tarL, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}
	s.SetWaitingTargets(tarL)
	s.Reallocate()

	cols = []string{"col-1", "col-2", "col-3"}
	s.SetCollectors(cols)
	s.ReallocateCollectors()
	for _, i := range s.collectors {
		assert.True(t, i.NumTargets == 5)
	}
}
