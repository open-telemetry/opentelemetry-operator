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

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var loggerPerNode = logf.Log.WithName("unit-tests")

// Tests that two targets with the same target url and job name but different label set are both added.
func TestAllocationPerNode(t *testing.T) {
	// prepare allocator with initial targets and collectors
	s, _ := New("per-node", loggerPerNode)

	cols := MakeNCollectors(3, 0)
	s.SetCollectors(cols)
	firstLabels := model.LabelSet{
		"test":           "test1",
		podNodeNameLabel: "node-0",
	}
	secondLabels := model.LabelSet{
		"test":           "test2",
		podNodeNameLabel: "node-1",
	}
	// no label, should be skipped
	thirdLabels := model.LabelSet{
		"test": "test3",
	}
	firstTarget := target.NewItem("sample-name", "0.0.0.0:8000", firstLabels, "")
	secondTarget := target.NewItem("sample-name", "0.0.0.0:8000", secondLabels, "")
	thirdTarget := target.NewItem("sample-name", "0.0.0.0:8000", thirdLabels, "")

	targetList := map[string]*target.Item{
		firstTarget.Hash():  firstTarget,
		secondTarget.Hash(): secondTarget,
		thirdTarget.Hash():  thirdTarget,
	}

	// test that targets and collectors are added properly
	s.SetTargets(targetList)

	// verify length
	actualItems := s.TargetItems()

	// one target should be skipped
	expectedTargetLen := len(targetList) - 1
	assert.Len(t, actualItems, expectedTargetLen)

	// verify allocation to nodes
	for targetHash, item := range targetList {
		actualItem, found := actualItems[targetHash]
		// if third target, should be skipped
		if targetHash != thirdTarget.Hash() {
			assert.True(t, found, "target with hash %s not found", item.Hash())
		} else {
			assert.False(t, found, "target with hash %s should not be found", item.Hash())
			return
		}

		itemsForCollector := s.GetTargetsForCollectorAndJob(actualItem.CollectorName, actualItem.JobName)
		assert.Len(t, itemsForCollector, 1)
		assert.Equal(t, actualItem, itemsForCollector[0])
	}
}
