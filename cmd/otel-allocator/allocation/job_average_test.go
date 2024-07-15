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

	"github.com/stretchr/testify/assert"
)

func TestCanSetCollectors(t *testing.T) {
	s, _ := New("job-average", logger)

	cols := MakeNCollectors(3, 0)
	s.SetCollectors(cols)

	expectedColLen := len(cols)
	collectors := s.Collectors()
	assert.Len(t, collectors, expectedColLen)

	for _, i := range cols {
		assert.NotNil(t, collectors[i.Name])
	}
}

func TestJobTargetsEvenDistribution(t *testing.T) {
	numCols := 15
	numItems := 1500
	numJobs := 10
	cols := MakeNCollectors(numCols, 0)
	c := newJobAverageAllocator(logger)
	c.SetCollectors(cols)
	c.SetTargets(MakeNNewTargetsInMJobs(numItems, numJobs, 0))
	actualTargetItems := c.TargetItems()
	expectedTargetsPerCollector := numItems / numCols
	expectedTargetsPerJobPerCollector := (numItems / numJobs) / numCols
	assert.Len(t, actualTargetItems, numItems)
	actualCollectors := c.Collectors()
	assert.Len(t, actualCollectors, numCols)
	for _, col := range actualCollectors {
		assert.Equal(t, col.NumTargets, expectedTargetsPerCollector)
	}
	collectorJobTargetCount := make(map[string]map[string]int)
	for _, item := range actualTargetItems {
		if jobCount, ok := collectorJobTargetCount[item.CollectorName]; ok {
			jobCount[item.JobName] = jobCount[item.JobName] + 1
		} else {
			cm := make(map[string]int)
			cm[item.JobName] = 1
			collectorJobTargetCount[item.CollectorName] = cm
		}
	}
	for _, m := range collectorJobTargetCount {
		for _, v := range m {
			assert.Equal(t, v, expectedTargetsPerJobPerCollector)
		}
	}

	// test collector reduce
	cols1 := MakeNCollectors(numCols-5, 0)
	c.SetCollectors(cols1)
	actualCollectors = c.Collectors()
	assert.Len(t, actualCollectors, numCols-5)
	actualTargetItems = c.TargetItems()
	collectorJobCount1 := make(map[string]map[string]int)
	expectedTargetsPerJobPerCollector1 := (numItems / numJobs) / (numCols - 5)
	for _, item := range actualTargetItems {
		if jobCount, ok := collectorJobCount1[item.CollectorName]; ok {
			jobCount[item.JobName] = jobCount[item.JobName] + 1
		} else {
			cm := make(map[string]int)
			cm[item.JobName] = 1
			collectorJobCount1[item.CollectorName] = cm
		}
	}
	for _, m := range collectorJobCount1 {
		for _, v := range m {
			assert.Equal(t, v, expectedTargetsPerJobPerCollector1)
		}
	}

}
