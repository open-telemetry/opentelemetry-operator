// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resourcekey

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    Key
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "without kind",
			args: args{
				key: "namespace/good",
			},
			want: Key{
				name:      "good",
				namespace: "namespace",
			},
			wantErr: assert.NoError,
		},
		{
			name: "with configmap kind",
			args: args{
				key: "configmap/namespace/good",
			},
			want: Key{
				name:      "good",
				namespace: "namespace",
				kind:      "configmap",
			},
			wantErr: assert.NoError,
		},
		{
			name: "with otelcol kind",
			args: args{
				key: "otelcol/namespace/good",
			},
			want: Key{
				name:      "good",
				namespace: "namespace",
				kind:      "otelcol",
			},
			wantErr: assert.NoError,
		},
		{
			name: "with unknown kind",
			args: args{
				key: "secret/namespace/good",
			},
			want: Key{
				name:      "good",
				namespace: "namespace",
				kind:      "secret",
			},
			wantErr: assert.NoError,
		},
		{
			name: "unable to get key",
			args: args{
				key: "badnamespace",
			},
			want:    Key{},
			wantErr: assert.Error,
		},
		{
			name: "too many slashes",
			args: args{
				key: "too/many/slashes/here",
			},
			want:    Key{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.key)
			if !tt.wantErr(t, err, fmt.Sprintf("Parse(%v)", tt.args.key)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Parse(%v)", tt.args.key)
		})
	}
}

func TestKey_String(t *testing.T) {
	type fields struct {
		name      string
		namespace string
		kind      string
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
		{
			name: "can make a key with kind",
			fields: fields{
				name:      "good",
				namespace: "namespace",
				kind:      KindConfigMap,
			},
			want: "configmap/namespace/good",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := New(tt.fields.namespace, tt.fields.name, tt.fields.kind)
			assert.Equalf(t, tt.want, k.String(), "String()")
		})
	}
}
