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
