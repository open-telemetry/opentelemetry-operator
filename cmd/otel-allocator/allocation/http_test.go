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
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var (
	baseLabelSet = model.LabelSet{
		"test-label": "test-value",
	}
	testJobLabelSetTwo = model.LabelSet{
		"test-label": "test-value2",
	}
	baseTargetItem       = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	testJobTargetItemTwo = target.NewItem("test-job", "test-url2", testJobLabelSetTwo, "test-collector2")
	secondTargetItem     = target.NewItem("test-job2", "test-url", baseLabelSet, "test-collector")
)

func TestGetAllTargetsByCollectorAndJob(t *testing.T) {
	leastWeighted, _ := New(leastWeightedStrategyName, logger)
	leastWeighted.SetCollectors(map[string]*Collector{"test-collector": {Name: "test-collector"}})
	type args struct {
		collector string
		job       string
		cMap      map[string][]target.Item
		allocator Allocator
	}
	var tests = []struct {
		name string
		args args
		want []target.Item
	}{
		{
			name: "Empty target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap:      map[string][]target.Item{},
				allocator: leastWeighted,
			},
			want: nil,
		},
		{
			name: "Single entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]target.Item{
					"test-collectortest-job": {
						*baseTargetItem,
					},
				},
				allocator: leastWeighted,
			},
			want: []target.Item{
				{
					TargetURL: []string{"test-url"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
					},
				},
			},
		},
		{
			name: "Multiple entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]target.Item{
					"test-collectortest-job": {
						*baseTargetItem,
					},
					"test-collectortest-job2": {
						*secondTargetItem,
					},
				},
				allocator: leastWeighted,
			},
			want: []target.Item{
				{
					TargetURL: []string{"test-url"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
					},
				},
			},
		},
		{
			name: "Multiple entry target map of same job with label merge",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]target.Item{
					"test-collectortest-job": {
						*baseTargetItem,
					},
					"test-collectortest-job2": {
						*testJobTargetItemTwo,
					},
				},
				allocator: leastWeighted,
			},
			want: []target.Item{
				{
					TargetURL: []string{"test-url1"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value-1",
						"foo":        "bar",
					},
				},
				{
					TargetURL: []string{"test-url2"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value-2",
					},
				},
			},
		},
		{
			name: "Multiple entry target map with same target address",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]target.Item{
					"test-collectortest-job": {
						*baseTargetItem,
						*baseTargetItem,
					},
				},
				allocator: leastWeighted,
			},
			want: []target.Item{
				{
					TargetURL: []string{"test-url"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
						"foo":        "bar",
					},
				},
				{
					TargetURL: []string{"test-url"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetGroups := GetAllTargetsByCollectorAndJob(tt.args.allocator, tt.args.collector, tt.args.job)
			for _, wantGroupJson := range tt.want {
				for _, groupJSON := range targetGroups {
					if groupJSON.Labels.String() == wantGroupJson.Labels.String() {
						assert.Equal(t, wantGroupJson, groupJSON)
					}
				}
			}
		})
	}
}

func BenchmarkGetAllTargetsByCollectorAndJob(b *testing.B) {
	var table = []struct {
		numCollectors int
		numJobs       int
	}{
		{numCollectors: 100, numJobs: 100},
		{numCollectors: 100, numJobs: 1000},
		{numCollectors: 100, numJobs: 10000},
		{numCollectors: 100, numJobs: 100000},
		{numCollectors: 1000, numJobs: 100},
		{numCollectors: 1000, numJobs: 1000},
		{numCollectors: 1000, numJobs: 10000},
		{numCollectors: 1000, numJobs: 100000},
	}
	for _, s := range GetRegisteredAllocatorNames() {
		for _, v := range table {
			a, err := New(s, logger)
			if err != nil {
				b.Log(err)
				b.Fail()
			}
			cols := makeNCollectors(v.numCollectors, 0)
			jobs := makeNNewTargets(v.numJobs, v.numCollectors, 0)
			a.SetCollectors(cols)
			a.SetTargets(jobs)
			b.Run(fmt.Sprintf("%s_num_cols_%d_num_jobs_%d", s, v.numCollectors, v.numJobs), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					GetAllTargetsByCollectorAndJob(a, fmt.Sprintf("collector-%d", v.numCollectors/2), fmt.Sprintf("test-job-%d", v.numJobs/2))
				}
			})
		}
	}
}
