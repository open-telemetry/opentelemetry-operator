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
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

func TestSetCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		expectedColLen := len(cols)
		collectors := allocator.Collectors()
		assert.Len(t, collectors, expectedColLen)

		for _, i := range cols {
			assert.NotNil(t, collectors[i.Name])
		}
	})
}

func TestSetTargets(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		targets := MakeNNewTargetsWithEmptyCollectors(3, 0)
		allocator.SetTargets(targets)

		expectedTargetLen := len(targets)
		actualTargets := allocator.TargetItems()
		assert.Len(t, actualTargets, expectedTargetLen)
	})
}

func TestCanSetSingleTarget(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		targets := MakeNNewTargetsWithEmptyCollectors(1, 3)
		allocator.SetCollectors(cols)
		allocator.SetTargets(targets)
		actualTargetItems := allocator.TargetItems()
		assert.Len(t, actualTargetItems, 1)
		for _, item := range actualTargetItems {
			assert.NotEmpty(t, item.CollectorName)
		}
	})
}

func TestCanSetTargetsBeforeCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		targets := MakeNNewTargetsWithEmptyCollectors(1, 3)
		allocator.SetTargets(targets)
		allocator.SetCollectors(cols)
		actualTargetItems := allocator.TargetItems()
		assert.Len(t, actualTargetItems, 1)
		for _, item := range actualTargetItems {
			assert.NotEmpty(t, item.CollectorName)
		}
	})
}

func TestAddingAndRemovingTargets(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)

		initTargets := MakeNNewTargets(6, 3, 0)

		// test that targets and collectors are added properly
		allocator.SetTargets(initTargets)

		// verify
		expectedTargetLen := len(initTargets)
		assert.Len(t, allocator.TargetItems(), expectedTargetLen)

		// prepare second round of targets
		tar := MakeNNewTargets(4, 3, 0)

		// test that fewer targets are found - removed
		allocator.SetTargets(tar)

		// verify
		targetItems := allocator.TargetItems()
		expectedNewTargetLen := len(tar)
		assert.Len(t, targetItems, expectedNewTargetLen)

		// verify results map
		for _, i := range tar {
			_, ok := targetItems[i.Hash()]
			assert.True(t, ok)
		}
	})
}

func TestAddingAndRemovingCollectors(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {
		targets := MakeNNewTargetsWithEmptyCollectors(3, 0)
		allocator.SetTargets(targets)

		collectors := MakeNCollectors(3, 0)

		// test that targets and collectors are added properly
		allocator.SetCollectors(collectors)

		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		for _, tg := range allocator.TargetItems() {
			if tg.CollectorName != "" {
				assert.Contains(t, collectors, tg.CollectorName)
			}
		}

		// remove a collector
		delete(collectors, "collector-0")
		allocator.SetCollectors(collectors)
		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		for _, tg := range allocator.TargetItems() {
			if tg.CollectorName != "" {
				assert.Contains(t, collectors, tg.CollectorName)
			}
		}

		// add two more collectors
		collectors = MakeNCollectors(5, 0)
		allocator.SetCollectors(collectors)

		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		for _, tg := range allocator.TargetItems() {
			if tg.CollectorName != "" {
				assert.Contains(t, collectors, tg.CollectorName)
			}
		}

		// remove all collectors
		collectors = map[string]*Collector{}
		allocator.SetCollectors(collectors)

		// verify
		assert.Len(t, allocator.Collectors(), len(collectors))
		jobs := []string{}
		for _, tg := range allocator.TargetItems() {
			assert.Empty(t, tg.CollectorName)
			jobs = append(jobs, tg.JobName)
		}
		for _, job := range jobs {
			for collector := range collectors {
				assert.Empty(t, allocator.GetTargetsForCollectorAndJob(collector, job))
			}
		}
	})
}

// Tests that two targets with the same target url and job name but different label set are both added.
func TestAllocationCollision(t *testing.T) {
	RunForAllStrategies(t, func(t *testing.T, allocator Allocator) {

		cols := MakeNCollectors(3, 0)
		allocator.SetCollectors(cols)
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
		allocator.SetTargets(targetList)

		// verify
		targetItems := allocator.TargetItems()
		expectedTargetLen := len(targetList)
		assert.Len(t, targetItems, expectedTargetLen)

		// verify results map
		for _, i := range targetList {
			_, ok := targetItems[i.Hash()]
			assert.True(t, ok)
		}
	})
}
