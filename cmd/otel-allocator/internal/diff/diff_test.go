// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diff

import (
	"reflect"
	"testing"
)

type HasherString string

func (s HasherString) Hash() string {
	return string(s)
}

func TestDiffMaps(t *testing.T) {
	type args struct {
		current map[string]Hasher
		new     map[string]Hasher
	}
	tests := []struct {
		name string
		args args
		want Changes[Hasher]
	}{
		{
			name: "basic replacement",
			args: args{
				current: map[string]Hasher{
					"current": HasherString("one"),
				},
				new: map[string]Hasher{
					"new": HasherString("another"),
				},
			},
			want: Changes[Hasher]{
				additions: map[string]Hasher{
					"new": HasherString("another"),
				},
				removals: map[string]Hasher{
					"current": HasherString("one"),
				},
			},
		},
		{
			name: "single addition",
			args: args{
				current: map[string]Hasher{
					"current": HasherString("one"),
				},
				new: map[string]Hasher{
					"current": HasherString("one"),
					"new":     HasherString("another"),
				},
			},
			want: Changes[Hasher]{
				additions: map[string]Hasher{
					"new": HasherString("another"),
				},
				removals: map[string]Hasher{},
			},
		},
		{
			name: "value change",
			args: args{
				current: map[string]Hasher{
					"k1":     HasherString("v1"),
					"k2":     HasherString("v2"),
					"change": HasherString("before"),
				},
				new: map[string]Hasher{
					"k1":     HasherString("v1"),
					"k3":     HasherString("v3"),
					"change": HasherString("after"),
				},
			},
			want: Changes[Hasher]{
				additions: map[string]Hasher{
					"k3":     HasherString("v3"),
					"change": HasherString("after"),
				},
				removals: map[string]Hasher{
					"k2":     HasherString("v2"),
					"change": HasherString("before"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Maps(tt.args.current, tt.args.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DiffMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}
