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
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func TestGetAllTargetsByCollectorAndJob(t *testing.T) {
	baseAllocator, _ := New("least-weighted", logger)
	baseAllocator.SetCollectors(map[string]*Collector{"test-collector": {Name: "test-collector"}})
	statefulAllocator, _ := New("least-weighted", logger)
	statefulAllocator.SetCollectors(map[string]*Collector{"test-collector-0": {Name: "test-collector-0"}})
	type args struct {
		collector string
		job       string
		cMap      map[string][]TargetItem
		allocator Allocator
	}
	var tests = []struct {
		name string
		args args
		want []targetGroupJSON
	}{
		{
			name: "Empty target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap:      map[string][]TargetItem{},
				allocator: baseAllocator,
			},
			want: nil,
		},
		{
			name: "Single entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]TargetItem{
					"test-collectortest-job": {
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL:     "test-url",
							CollectorName: "test-collector",
						},
					},
				},
				allocator: baseAllocator,
			},
			want: []targetGroupJSON{
				{
					Targets: []string{"test-url"},
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
				cMap: map[string][]TargetItem{
					"test-collectortest-job": {
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL:     "test-url",
							CollectorName: "test-collector",
						},
					},
					"test-collectortest-job2": {
						TargetItem{
							JobName: "test-job2",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL:     "test-url",
							CollectorName: "test-collector",
						},
					},
				},
				allocator: baseAllocator,
			},
			want: []targetGroupJSON{
				{
					Targets: []string{"test-url"},
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
				cMap: map[string][]TargetItem{
					"test-collectortest-job": {
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
								"foo":        "bar",
							},
							TargetURL:     "test-url1",
							CollectorName: "test-collector",
						},
					},
					"test-collectortest-job2": {
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL:     "test-url2",
							CollectorName: "test-collector",
						},
					},
				},
				allocator: baseAllocator,
			},
			want: []targetGroupJSON{
				{
					Targets: []string{"test-url1"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
						"foo":        "bar",
					},
				},
				{
					Targets: []string{"test-url2"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
					},
				},
			},
		},
		{
			name: "Multiple entry target map with same target address",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]TargetItem{
					"test-collectortest-job": {
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
								"foo":        "bar",
							},
							TargetURL:     "test-url",
							CollectorName: "test-collector",
						},
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL:     "test-url",
							CollectorName: "test-collector",
						},
					},
				},
				allocator: baseAllocator,
			},
			want: []targetGroupJSON{
				{
					Targets: []string{"test-url"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
						"foo":        "bar",
					},
				},
				{
					Targets: []string{"test-url"},
					Labels: map[model.LabelName]model.LabelValue{
						"test-label": "test-value",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetGroups := GetAllTargetsByCollectorAndJob(tt.args.collector, tt.args.job, tt.args.cMap, tt.args.allocator)
			for _, wantGroupJson := range tt.want {
				exist := false
				for _, groupJSON := range targetGroups {
					if groupJSON.Labels.String() == wantGroupJson.Labels.String() {
						exist = reflect.DeepEqual(groupJSON, wantGroupJson)
					}
				}
				assert.Equal(t, true, exist)
			}
		})
	}
}
