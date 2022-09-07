package allocation

import (
	"reflect"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	_ "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/least_weighted"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/strategy"
)

var logger = logf.Log.WithName("unit-tests")

func TestGetAllTargetsByCollectorAndJob(t *testing.T) {
	baseAllocator, _ := strategy.New("least-weighted", logger)
	baseAllocator.SetCollectors(map[string]*strategy.Collector{"test-collector": {Name: "test-collector"}})
	statefulAllocator, _ := strategy.New("least-weighted", logger)
	statefulAllocator.SetCollectors(map[string]*strategy.Collector{"test-collector-0": {Name: "test-collector-0"}})
	type args struct {
		collector string
		job       string
		cMap      map[string][]strategy.TargetItem
		allocator strategy.Allocator
	}
	var tests = []struct {
		name string
		args args
		want []strategy.TargetGroupJSON
	}{
		{
			name: "Empty target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap:      map[string][]strategy.TargetItem{},
				allocator: baseAllocator,
			},
			want: nil,
		},
		{
			name: "Single entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string][]strategy.TargetItem{
					"test-collectortest-job": {
						strategy.TargetItem{
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
			want: []strategy.TargetGroupJSON{
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
				cMap: map[string][]strategy.TargetItem{
					"test-collectortest-job": {
						strategy.TargetItem{
							JobName: "test-job",
							Label: model.LabelSet{
								"test-label": "test-value",
							},
							TargetURL:     "test-url",
							CollectorName: "test-collector",
						},
					},
					"test-collectortest-job2": {
						strategy.TargetItem{
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
			want: []strategy.TargetGroupJSON{
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
				cMap: map[string][]strategy.TargetItem{
					"test-collectortest-job": {
						strategy.TargetItem{
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
						strategy.TargetItem{
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
			want: []strategy.TargetGroupJSON{
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
