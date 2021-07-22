package allocation

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

// Tests least connection - The expected collector after running findNextCollector should be the collecter with the least amount of workload
func TestFindNextSharder(t *testing.T) {
	s := NewAllocator()
	defaultCol := collector{Name: "default-col", NumTargets: 1}
	maxCol := collector{Name: "max-col", NumTargets: 2}
	leastCol := collector{Name: "least-col", NumTargets: 0}
	s.collectors[maxCol.Name] = &maxCol
	s.collectors[leastCol.Name] = &leastCol
	s.nextCollector = &defaultCol

	s.findNextCollector()
	assert.Equal(t, "least-col", s.nextCollector.Name)
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
	// prepare lb with initial targets and collectors
	s := NewAllocator()
	cols := []string{"col-1", "col-2", "col-3"}
	initTargets := []string{"targ:1000", "targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005"}
	s.SetCollectors(cols)
	var targetList []TargetItem
	for _, i := range initTargets {
		targetList = append(targetList, TargetItem{JobName: "sample-name", TargetURL: i, Label: model.LabelSet{}})
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)
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
	s.SetTargets(tarL)
	s.Reallocate()

	// verify
	assert.True(t, len(s.targetItems) == 4)

	// verify results map
	for _, i := range tar {
		_, ok := s.targetItems["sample-name"+i]
		assert.True(t, ok)
	}
}
