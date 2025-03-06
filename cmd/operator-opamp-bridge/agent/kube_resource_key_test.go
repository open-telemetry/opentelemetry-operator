// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
		want    kubeResourceKey
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "base case",
			args: args{
				key: "namespace/good",
			},
			want: kubeResourceKey{
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
			want:    kubeResourceKey{},
			wantErr: assert.Error,
		},
		{
			name: "too many slashes",
			args: args{
				key: "too/many/slashes",
			},
			want:    kubeResourceKey{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kubeResourceFromKey(tt.args.key)
			if !tt.wantErr(t, err, fmt.Sprintf("kubeResourceFromKey(%v)", tt.args.key)) {
				return
			}
			assert.Equalf(t, tt.want, got, "kubeResourceFromKey(%v)", tt.args.key)
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
			k := newKubeResourceKey(tt.fields.namespace, tt.fields.name)
			assert.Equalf(t, tt.want, k.String(), "String()")
		})
	}
}
