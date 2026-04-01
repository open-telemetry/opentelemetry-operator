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
		current map[string]Hasher[string]
		new     map[string]Hasher[string]
	}
	tests := []struct {
		name string
		args args
		want Changes[string, Hasher[string]]
	}{
		{
			name: "basic replacement",
			args: args{
				current: map[string]Hasher[string]{
					"current": HasherString("one"),
				},
				new: map[string]Hasher[string]{
					"new": HasherString("another"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{
					"new": HasherString("another"),
				},
				removals: map[string]Hasher[string]{
					"current": HasherString("one"),
				},
			},
		},
		{
			name: "single addition",
			args: args{
				current: map[string]Hasher[string]{
					"current": HasherString("one"),
				},
				new: map[string]Hasher[string]{
					"current": HasherString("one"),
					"new":     HasherString("another"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{
					"new": HasherString("another"),
				},
				removals: map[string]Hasher[string]{},
			},
		},
		{
			name: "value change",
			args: args{
				current: map[string]Hasher[string]{
					"k1":     HasherString("v1"),
					"k2":     HasherString("v2"),
					"change": HasherString("before"),
				},
				new: map[string]Hasher[string]{
					"k1":     HasherString("v1"),
					"k3":     HasherString("v3"),
					"change": HasherString("after"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{
					"k3":     HasherString("v3"),
					"change": HasherString("after"),
				},
				removals: map[string]Hasher[string]{
					"k2":     HasherString("v2"),
					"change": HasherString("before"),
				},
			},
		},
		{
			name: "both empty maps",
			args: args{
				current: map[string]Hasher[string]{},
				new:     map[string]Hasher[string]{},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{},
				removals:  map[string]Hasher[string]{},
			},
		},
		{
			name: "empty current, non-empty new",
			args: args{
				current: map[string]Hasher[string]{},
				new: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
				},
				removals: map[string]Hasher[string]{},
			},
		},
		{
			name: "non-empty current, empty new",
			args: args{
				current: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
				},
				new: map[string]Hasher[string]{},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{},
				removals: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
				},
			},
		},
		{
			name: "identical maps",
			args: args{
				current: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
					"c": HasherString("3"),
				},
				new: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
					"c": HasherString("3"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{},
				removals:  map[string]Hasher[string]{},
			},
		},
		{
			name: "same key different hash",
			args: args{
				current: map[string]Hasher[string]{
					"k": HasherString("hash-v1"),
				},
				new: map[string]Hasher[string]{
					"k": HasherString("hash-v2"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{
					"k": HasherString("hash-v2"),
				},
				removals: map[string]Hasher[string]{
					"k": HasherString("hash-v1"),
				},
			},
		},
		{
			name: "complete swap of all entries",
			args: args{
				current: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
				},
				new: map[string]Hasher[string]{
					"c": HasherString("3"),
					"d": HasherString("4"),
				},
			},
			want: Changes[string, Hasher[string]]{
				additions: map[string]Hasher[string]{
					"c": HasherString("3"),
					"d": HasherString("4"),
				},
				removals: map[string]Hasher[string]{
					"a": HasherString("1"),
					"b": HasherString("2"),
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

func TestNewChanges(t *testing.T) {
	additions := map[string]Hasher[string]{"a": HasherString("1")}
	removals := map[string]Hasher[string]{"b": HasherString("2")}
	c := NewChanges(additions, removals)
	if !reflect.DeepEqual(c.Additions(), additions) {
		t.Errorf("Additions() = %v, want %v", c.Additions(), additions)
	}
	if !reflect.DeepEqual(c.Removals(), removals) {
		t.Errorf("Removals() = %v, want %v", c.Removals(), removals)
	}
}

func TestNewChangesNil(t *testing.T) {
	c := NewChanges[string, Hasher[string]](nil, nil)
	if c.Additions() != nil {
		t.Errorf("expected nil additions, got %v", c.Additions())
	}
	if c.Removals() != nil {
		t.Errorf("expected nil removals, got %v", c.Removals())
	}
}

// HasherInt tests that the generic diff works with a non-string key type.
type HasherInt int

func (h HasherInt) Hash() int {
	return int(h)
}

func TestDiffMapsIntKey(t *testing.T) {
	current := map[int]Hasher[int]{
		1: HasherInt(10),
		2: HasherInt(20),
	}
	updated := map[int]Hasher[int]{
		2: HasherInt(20),
		3: HasherInt(30),
	}
	got := Maps(current, updated)
	wantAdditions := map[int]Hasher[int]{3: HasherInt(30)}
	wantRemovals := map[int]Hasher[int]{1: HasherInt(10)}
	if !reflect.DeepEqual(got.Additions(), wantAdditions) {
		t.Errorf("Additions() = %v, want %v", got.Additions(), wantAdditions)
	}
	if !reflect.DeepEqual(got.Removals(), wantRemovals) {
		t.Errorf("Removals() = %v, want %v", got.Removals(), wantRemovals)
	}
}
