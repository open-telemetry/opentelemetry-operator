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

package agent

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_collectorKeyFromKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    collectorKey
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "base case",
			args: args{
				key: "namespace/good",
			},
			want: collectorKey{
				name:      "good",
				namespace: "namespace",
			},
			wantErr: assert.NoError,
		},
		{
			name: "unable to get key",
			args: args{
				key: "badnamespace",
			},
			want:    collectorKey{},
			wantErr: assert.Error,
		},
		{
			name: "too many slashes",
			args: args{
				key: "too/many/slashes",
			},
			want:    collectorKey{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collectorKeyFromKey(tt.args.key)
			if !tt.wantErr(t, err, fmt.Sprintf("collectorKeyFromKey(%v)", tt.args.key)) {
				return
			}
			assert.Equalf(t, tt.want, got, "collectorKeyFromKey(%v)", tt.args.key)
		})
	}
}

func Test_collectorKey_String(t *testing.T) {
	type fields struct {
		name      string
		namespace string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "can make a key",
			fields: fields{
				name:      "good",
				namespace: "namespace",
			},
			want: "namespace/good",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newCollectorKey(tt.fields.namespace, tt.fields.name)
			assert.Equalf(t, tt.want, k.String(), "String()")
		})
	}
}
