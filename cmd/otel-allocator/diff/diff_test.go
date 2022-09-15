package diff

import (
	"reflect"
	"testing"
)

func TestDiffMaps(t *testing.T) {
	type args struct {
		current map[string]string
		new     map[string]string
	}
	tests := []struct {
		name string
		args args
		want Changes[string]
	}{
		{
			name: "basic replacement",
			args: args{
				current: map[string]string{
					"current": "one",
				},
				new: map[string]string{
					"new": "another",
				},
			},
			want: Changes[string]{
				additions: map[string]string{
					"new": "another",
				},
				removals: map[string]string{
					"current": "one",
				},
			},
		},
		{
			name: "single addition",
			args: args{
				current: map[string]string{
					"current": "one",
				},
				new: map[string]string{
					"current": "one",
					"new":     "another",
				},
			},
			want: Changes[string]{
				additions: map[string]string{
					"new": "another",
				},
				removals: map[string]string{},
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
