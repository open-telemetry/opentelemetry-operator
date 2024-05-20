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

// Note: These utilities are used by other packages, which is why they're defined in a non-test file.

package allocation

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

func colIndex(index, numCols int) int {
	if numCols == 0 {
		return -1
	}
	return index % numCols
}

func MakeNNewTargets(n int, numCollectors int, startingIndex int) map[string]*target.Item {
	toReturn := map[string]*target.Item{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", colIndex(i, numCollectors))
		label := model.LabelSet{
			"i":     model.LabelValue(strconv.Itoa(i)),
			"total": model.LabelValue(strconv.Itoa(n + startingIndex)),
		}
		newTarget := target.NewItem(fmt.Sprintf("test-job-%d", i), fmt.Sprintf("test-url-%d", i), label, collector)
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func MakeNCollectors(n int, startingIndex int) map[string]*Collector {
	toReturn := map[string]*Collector{}
	for i := startingIndex; i < n+startingIndex; i++ {
		collector := fmt.Sprintf("collector-%d", i)
		toReturn[collector] = &Collector{
			Name:       collector,
			NumTargets: 0,
			NodeName:   fmt.Sprintf("node-%d", i),
		}
	}
	return toReturn
}

func MakeNNewTargetsWithEmptyCollectors(n int, startingIndex int) map[string]*target.Item {
	toReturn := map[string]*target.Item{}
	for i := startingIndex; i < n+startingIndex; i++ {
		label := model.LabelSet{
			"i":                               model.LabelValue(strconv.Itoa(i)),
			"total":                           model.LabelValue(strconv.Itoa(n + startingIndex)),
			"__meta_kubernetes_pod_node_name": model.LabelValue("node-0"),
		}
		newTarget := target.NewItem(fmt.Sprintf("test-job-%d", i), fmt.Sprintf("test-url-%d", i), label, "")
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func RunForAllStrategies(t *testing.T, f func(t *testing.T, allocator Allocator)) {
	allocatorNames := GetRegisteredAllocatorNames()
	logger := logf.Log.WithName("unit-tests")
	for _, allocatorName := range allocatorNames {
		t.Run(allocatorName, func(t *testing.T) {
			allocator, err := New(allocatorName, logger)
			require.NoError(t, err)
			f(t, allocator)
		})
	}
}
