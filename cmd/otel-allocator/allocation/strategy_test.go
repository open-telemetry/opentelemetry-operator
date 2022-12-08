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
	"reflect"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/diff"
)

func Benchmark_Setting(b *testing.B) {
	var table = []struct {
		numCollectors int
		numTargets    int
	}{
		{numCollectors: 100, numTargets: 100},
		{numCollectors: 100, numTargets: 1000},
		{numCollectors: 100, numTargets: 10000},
		{numCollectors: 100, numTargets: 100000},
		{numCollectors: 1000, numTargets: 100},
		{numCollectors: 1000, numTargets: 1000},
		{numCollectors: 1000, numTargets: 10000},
		{numCollectors: 1000, numTargets: 100000},
	}

	for _, s := range GetRegisteredAllocatorNames() {
		for _, v := range table {
			// prepare allocator with 3 collectors and 'random' amount of targets
			a, _ := New(s, logger)
			cols := makeNCollectors(v.numCollectors, 0)
			targets := makeNNewTargets(v.numTargets, v.numCollectors, 0)
			b.Run(fmt.Sprintf("%s_num_cols_%d_num_jobs_%d", s, v.numCollectors, v.numTargets), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					a.SetCollectors(cols)
					a.SetTargets(targets)
				}
			})
		}
	}
}

func TestCollectorDiff(t *testing.T) {
	collector0 := NewCollector("collector-0")
	collector1 := NewCollector("collector-1")
	collector2 := NewCollector("collector-2")
	collector3 := NewCollector("collector-3")
	collector4 := NewCollector("collector-4")
	type args struct {
		current map[string]*Collector
		new     map[string]*Collector
	}
	tests := []struct {
		name string
		args args
		want diff.Changes[*Collector]
	}{
		{
			name: "diff two collector maps",
			args: args{
				current: map[string]*Collector{
					"collector-0": collector0,
					"collector-1": collector1,
					"collector-2": collector2,
					"collector-3": collector3,
				},
				new: map[string]*Collector{
					"collector-0": collector0,
					"collector-1": collector1,
					"collector-2": collector2,
					"collector-4": collector4,
				},
			},
			want: diff.NewChanges(map[string]*Collector{
				"collector-4": collector4,
			}, map[string]*Collector{
				"collector-3": collector3,
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := diff.Maps(tt.args.current, tt.args.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DiffMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}
