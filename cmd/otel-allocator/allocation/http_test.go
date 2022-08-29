package allocation

import (
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func TestGetAllTargetsByCollectorAndJob(t *testing.T) {
	baseAllocator := NewAllocator(logger)
	baseAllocator.SetCollectors([]string{"test-collector"})
	statefulAllocator := NewAllocator(logger)
	statefulAllocator.SetCollectors([]string{"test-collector-0"})
	type args struct {
		collector string
		job       string
		cMap      map[string][]TargetItem
		allocator *Allocator
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
							TargetURL: "test-url",
							Collector: &collector{
								Name:       "test-collector",
								NumTargets: 1,
							},
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
							TargetURL: "test-url",
							Collector: &collector{
								Name:       "test-collector",
								NumTargets: 1,
							},
						},
					},
					"test-collectortest-job2": {
						TargetItem{
							JobName: "test-job2",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL: "test-url",
							Collector: &collector{
								Name:       "test-collector",
								NumTargets: 1,
							},
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
							TargetURL: "test-url1",
							Collector: &collector{
								Name:       "test-collector",
								NumTargets: 2,
							},
						},
					},
					"test-collectortest-job2": {
						TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL: "test-url2",
							Collector: &collector{
								Name:       "test-collector",
								NumTargets: 2,
							},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, groupJSON := range GetAllTargetsByCollectorAndJob(tt.args.collector, tt.args.job, tt.args.cMap, tt.args.allocator) {
				exist := false
				for _, wantGroupJson := range tt.want {
					if groupJSON.Labels.String() == wantGroupJson.Labels.String() {
						exist = reflect.DeepEqual(groupJSON, wantGroupJson)
					}
				}
				assert.Equal(t, true, exist)
			}
		})
	}
}
