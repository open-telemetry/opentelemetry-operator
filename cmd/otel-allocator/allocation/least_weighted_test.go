// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allocation

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var logger = logf.Log.WithName("unit-tests")

func colIndex(index, numCols int) int {
	if numCols == 0 {
		return -1
	}
	return index % numCols
}

func makeNNewTargets(n int, numCollectors int, startingIndex int) map[string]*target.Item {
	toReturn := map[string]*target.Item{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		label := model.LabelSet{
			"collector": model.LabelValue(collector),
			"i":         model.LabelValue(strconv.Itoa(i)),
			"total":     model.LabelValue(strconv.Itoa(n + startingIndex)),
		}
		newTarget := target.NewItem(fmt.Sprintf("test-job-%d", i), "test-url", label, collector)
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func makeNCollectors(n int, startingIndex int) map[string]*Collector {
	toReturn := map[string]*Collector{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", i)
		toReturn[collector] = &Collector{
			Name:       collector,
			NumTargets: 0,
		}
	}
	return toReturn
}

func TestSetCollectors(t *testing.T) {
	s, _ := New("least-weighted", logger)

	cols := makeNCollectors(3, 0)
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	collectors := s.Collectors()
	assert.Len(t, collectors, expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, collectors[i.Name])
	}
}

func TestAddingAndRemovingTargets(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s, _ := New("least-weighted", logger)

	cols := makeNCollectors(3, 0)
	s.SetCollectors(cols)

	initTargets := makeNNewTargets(6, 3, 0)

	// test that targets and collectors are added properly
	s.SetTargets(initTargets)

	// verify
	expectedTargetLen := len(initTargets)
	assert.Len(t, s.TargetItems(), expectedTargetLen)

	// prepare second round of targets
	tar := makeNNewTargets(4, 3, 0)

	// test that fewer targets are found - removed
	s.SetTargets(tar)

	// verify
	targetItems := s.TargetItems()
	expectedNewTargetLen := len(tar)
	assert.Len(t, targetItems, expectedNewTargetLen)

	// verify results map
	for _, i := range tar {
		_, ok := targetItems[i.Hash()]
		assert.True(t, ok)
	}
}

// Tests that two targets with the same target url and job name but different label set are both added.
func TestAllocationCollision(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s, _ := New("least-weighted", logger)

	cols := makeNCollectors(3, 0)
	s.SetCollectors(cols)
	firstLabels := model.LabelSet{
		"test": "test1",
	}
	secondLabels := model.LabelSet{
		"test": "test2",
	}
	firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstLabels, "")
	secondTarget := target.NewItem("sample-name", "0.0.0.0:8000", secondLabels, "")

	targetList := map[string]*target.Item{
		firstTarget.Hash():  firstTarget,
		secondTarget.Hash(): secondTarget,
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify
	targetItems := s.TargetItems()
	expectedTargetLen := len(targetList)
	assert.Len(t, targetItems, expectedTargetLen)

	// verify results map
	for _, i := range targetList {
		_, ok := targetItems[i.Hash()]
		assert.True(t, ok)
	}
}

func TestNoCollectorReassignment(t *testing.T) {
	s, _ := New("least-weighted", logger)

	cols := makeNCollectors(3, 0)
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	assert.Len(t, s.Collectors(), expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.Collectors()[i.Name])
	}
	initTargets := makeNNewTargets(6, 3, 0)

	// test that targets and collectors are added properly
	s.SetTargets(initTargets)

	// verify
	expectedTargetLen := len(initTargets)
	targetItems := s.TargetItems()
	assert.Len(t, targetItems, expectedTargetLen)

	// assign new set of collectors with the same names
	newCols := makeNCollectors(3, 0)
	s.SetCollectors(newCols)

	newTargetItems := s.TargetItems()
	assert.Equal(t, targetItems, newTargetItems)

}

func TestSmartCollectorReassignment(t *testing.T) {
	t.Skip("This test is flaky and fails frequently, see issue 1291")
	s, _ := New("least-weighted", logger)

	cols := makeNCollectors(4, 0)
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	assert.Len(t, s.Collectors(), expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, s.Collectors()[i.Name])
	}
	initTargets := makeNNewTargets(6, 0, 0)
	// test that targets and collectors are added properly
	s.SetTargets(initTargets)

	// verify
	expectedTargetLen := len(initTargets)
	targetItems := s.TargetItems()
	assert.Len(t, targetItems, expectedTargetLen)

	// assign new set of collectors with the same names
	newCols := map[string]*Collector{
		"collector-0": {
			Name: "collector-0",
		}, "collector-1": {
			Name: "collector-1",
		}, "collector-2": {
			Name: "collector-2",
		}, "collector-4": {
			Name: "collector-4",
		},
	}
	s.SetCollectors(newCols)

	newTargetItems := s.TargetItems()
	assert.Equal(t, len(targetItems), len(newTargetItems))
	for key, targetItem := range targetItems {
		item, ok := newTargetItems[key]
		assert.True(t, ok, "all target items should be found in new target item list")
		if targetItem.CollectorName != "collector-3" {
			assert.Equal(t, targetItem.CollectorName, item.CollectorName)
		} else {
			assert.Equal(t, "collector-4", item.CollectorName)
		}
	}
}

// Tests that the delta in number of targets per collector is less than 15% of an even distribution.
func TestCollectorBalanceWhenAddingAndRemovingAtRandom(t *testing.T) {

	// prepare allocator with 3 collectors and 'random' amount of targets
	s, _ := New("least-weighted", logger)

	cols := makeNCollectors(3, 0)
	s.SetCollectors(cols)

	targets := makeNNewTargets(27, 3, 0)
	s.SetTargets(targets)

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
	// Remove half of targets randomly
	toDelete := len(targets) / 2
	counter := 0
	for index := range targets {
		shouldDelete := rand.Intn(toDelete) //nolint:gosec
		if counter < shouldDelete {
			delete(targets, index)
		}
		counter++
	}
	s.SetTargets(targets)

	targetItemLen = len(s.TargetItems())
	collectors = s.Collectors()
	count = targetItemLen / len(collectors)
	percent = float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
	// adding targets at 'random'
	for _, item := range makeNNewTargets(13, 3, 100) {
		targets[item.Hash()] = item
	}
	s.SetTargets(targets)

	targetItemLen = len(s.TargetItems())
	collectors = s.Collectors()
	count = targetItemLen / len(collectors)
	percent = float64(targetItemLen) / divisor

	// test
	for _, i := range collectors {
		assert.InDelta(t, i.NumTargets, count, math.Round(percent))
	}
}
