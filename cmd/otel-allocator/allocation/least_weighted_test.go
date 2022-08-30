package allocation

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func makeNNewTargets(n int, numCollectors int) map[string]TargetItem {
	toReturn := map[string]TargetItem{}
	for i := 0; i < n; i++ {
		collector := fmt.Sprintf("collector-%d", i%numCollectors)
		newTarget := NewTargetItem(fmt.Sprintf("test-job-%d", i), "test-url", nil, collector)
		toReturn[newTarget.hash()] = newTarget
	}
	return toReturn
}

func TestLeastWeightedStrategy_Allocate(t *testing.T) {
	type args struct {
		currentState State
		newState     State
	}
	tests := []struct {
		name string
		args args
		want State
	}{
		{
			name: "single collector gets a new target",
			args: args{
				currentState: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 0,
						},
					},
					targetItems: map[string]TargetItem{},
				},
				newState: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 0,
						},
					},
					targetItems: makeNNewTargets(1, 1),
				},
			},
			want: State{
				collectors: map[string]collector{
					"collector-0": {
						Name:       "collector-0",
						NumTargets: 1,
					},
				},
				targetItems: makeNNewTargets(1, 1),
			},
		},
		{
			name: "test set new collectors",
			args: args{
				currentState: State{
					collectors:  map[string]collector{},
					targetItems: map[string]TargetItem{},
				},
				newState: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 0,
						},
						"collector-1": {
							Name:       "collector-1",
							NumTargets: 0,
						},
						"collector-2": {
							Name:       "collector-2",
							NumTargets: 0,
						},
					},
					targetItems: map[string]TargetItem{},
				},
			},
			want: State{
				collectors: map[string]collector{
					"collector-0": {
						Name:       "collector-0",
						NumTargets: 0,
					},
					"collector-1": {
						Name:       "collector-1",
						NumTargets: 0,
					},
					"collector-2": {
						Name:       "collector-2",
						NumTargets: 0,
					},
				},
				targetItems: map[string]TargetItem{},
			},
		},
		{
			name: "test remove targets",
			args: args{
				currentState: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 2,
						},
						"collector-1": {
							Name:       "collector-1",
							NumTargets: 2,
						},
					},
					targetItems: makeNNewTargets(4, 2),
				},
				newState: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 2,
						},
						"collector-1": {
							Name:       "collector-1",
							NumTargets: 2,
						},
					},
					targetItems: makeNNewTargets(2, 2),
				},
			},
			want: State{
				collectors: map[string]collector{
					"collector-0": {
						Name:       "collector-0",
						NumTargets: 1,
					},
					"collector-1": {
						Name:       "collector-1",
						NumTargets: 1,
					},
				},
				targetItems: makeNNewTargets(2, 2),
			},
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
		state State
	}
	tests := []struct {
		name string
		args args
		want collector
	}{
		{
			name: "goes to first collector with no targets",
			args: args{
				state: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 0,
						},
						"collector-1": {
							Name:       "collector-1",
							NumTargets: 0,
						},
					},
					targetItems: nil,
				},
			},
			want: collector{
				Name:       "collector-0",
				NumTargets: 0,
			},
		},
		{
			name: "goes to collector with fewest targets with existing state",
			args: args{
				state: State{
					collectors: map[string]collector{
						"collector-0": {
							Name:       "collector-0",
							NumTargets: 0,
						},
						"collector-1": {
							Name:       "collector-1",
							NumTargets: 0,
						},
						"collector-2": {
							Name:       "collector-2",
							NumTargets: 2,
						},
					},
					targetItems: nil,
				},
			},
			want: collector{
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
	emptyState := State{
		collectors:  map[string]collector{},
		targetItems: map[string]TargetItem{},
	}
	for i := 0; i < b.N; i++ {
		l.Allocate(emptyState, State{
			collectors: map[string]collector{
				"collector-0": {
					Name:       "collector-0",
					NumTargets: 0,
				},
				"collector-1": {
					Name:       "collector-1",
					NumTargets: 0,
				},
				"collector-2": {
					Name:       "collector-2",
					NumTargets: 0,
				},
			},
			targetItems: makeNNewTargets(i, 3),
		})
	}
}
