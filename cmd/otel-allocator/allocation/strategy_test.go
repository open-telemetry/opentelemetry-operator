// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/diff"
)

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
			cols := MakeNCollectors(v.numCollectors, 0)
			jobs := MakeNNewTargets(v.numJobs, v.numCollectors, 0)
			a.SetCollectors(cols)
			a.SetTargets(jobs)
			b.Run(fmt.Sprintf("%s_num_cols_%d_num_jobs_%d", s, v.numCollectors, v.numJobs), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					a.GetTargetsForCollectorAndJob(fmt.Sprintf("collector-%d", v.numCollectors/2), fmt.Sprintf("test-job-%d", v.numJobs/2))
				}
			})
		}
	}
}

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
			a, _ := New(s, logger)
			cols := MakeNCollectors(v.numCollectors, 0)
			targets := MakeNNewTargets(v.numTargets, v.numCollectors, 0)
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
	collector0 := NewCollector("collector-0", "")
	collector1 := NewCollector("collector-1", "")
	collector2 := NewCollector("collector-2", "")
	collector3 := NewCollector("collector-3", "")
	collector4 := NewCollector("collector-4", "")
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
