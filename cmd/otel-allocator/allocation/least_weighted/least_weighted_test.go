package least_weighted

import (
	"fmt"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/strategy"

	"github.com/stretchr/testify/assert"
)

func makeNNewTargets(n int, numCollectors int) map[string]strategy.TargetItem {
	toReturn := map[string]strategy.TargetItem{}
	for i := 0; i < n; i++ {
		collector := fmt.Sprintf("collector-%d", i%numCollectors)
		newTarget := strategy.NewTargetItem(fmt.Sprintf("test-job-%d", i), "test-url", nil, collector)
		toReturn[newTarget.Hash()] = newTarget
	}
	return toReturn
}

func makeNCollectors(n int, targetsForEach int) map[string]strategy.Collector {
	toReturn := map[string]strategy.Collector{}
	for i := 0; i < n; i++ {
		collector := fmt.Sprintf("collector-%d", i)
		toReturn[collector] = strategy.Collector{
			Name:       collector,
			NumTargets: targetsForEach,
		}
	}
	return toReturn
}

func TestLeastWeightedStrategy_Allocate(t *testing.T) {
	type args struct {
		currentState strategy.State
		newState     strategy.State
	}
	tests := []struct {
		name string
		args args
		want strategy.State
	}{
		{
			name: "single collector gets a new target",
			args: args{
				currentState: strategy.NewState(makeNCollectors(1, 0), makeNNewTargets(0, 1)),
				newState:     strategy.NewState(makeNCollectors(1, 0), makeNNewTargets(1, 1)),
			},
			want: strategy.NewState(makeNCollectors(1, 1), makeNNewTargets(1, 1)),
		},
		{
			name: "test set new collectors",
			args: args{
				currentState: strategy.NewState(makeNCollectors(0, 0), makeNNewTargets(0, 0)),
				newState:     strategy.NewState(makeNCollectors(3, 0), makeNNewTargets(0, 3)),
			},
			want: strategy.NewState(makeNCollectors(3, 0), makeNNewTargets(0, 3)),
		},
		{
			name: "test remove targets",
			args: args{
				currentState: strategy.NewState(makeNCollectors(2, 2), makeNNewTargets(4, 2)),
				newState:     strategy.NewState(makeNCollectors(2, 2), makeNNewTargets(2, 2)),
			},
			want: strategy.NewState(makeNCollectors(2, 1), makeNNewTargets(2, 2)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LeastWeightedStrategy{}
			assert.Equalf(t, tt.want, l.Allocate(tt.args.currentState, tt.args.newState), "Allocate(%v, %v)", tt.args.currentState, tt.args.newState)
		})
	}
}

func TestLeastWeightedStrategy_findNextCollector(t *testing.T) {
	type args struct {
		state strategy.State
	}
	tests := []struct {
		name string
		args args
		want strategy.Collector
	}{
		{
			name: "goes to first collector with no targets",
			args: args{
				state: strategy.NewState(makeNCollectors(1, 0), makeNNewTargets(0, 1)),
			},
			want: strategy.Collector{
				Name:       "collector-0",
				NumTargets: 0,
			},
		},
		{
			name: "goes to collector with fewest targets with existing state",
			args: args{
				state: strategy.NewState(
					map[string]strategy.Collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 0,
						},
						"collector-1": {
							Name:       "collector-1",
							NumTargets: 1,
						},
						"collector-2": {
							Name:       "collector-2",
							NumTargets: 2,
						},
					},
					nil,
				),
			},
			want: strategy.Collector{
				Name:       "collector-0",
				NumTargets: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LeastWeightedStrategy{}
			assert.Equalf(t, tt.want, l.findNextCollector(tt.args.state), "findNextCollector(%v)", tt.args.state)
		})
	}
}

func BenchmarkLeastWeightedStrategy_AllocateTargets(b *testing.B) {
	l := LeastWeightedStrategy{}
	emptyState := strategy.NewState(map[string]strategy.Collector{}, map[string]strategy.TargetItem{})
	for i := 0; i < b.N; i++ {
		l.Allocate(emptyState, strategy.NewState(makeNCollectors(3, 0), makeNNewTargets(i, 3)))
	}
}
