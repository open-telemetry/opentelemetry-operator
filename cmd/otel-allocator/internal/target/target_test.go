// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemHash_String(t *testing.T) {
	tests := []struct {
		name string
		h    ItemHash
		want string
	}{
		{
			name: "empty",
			h:    0,
			want: "0",
		},
		{
			name: "non-empty",
			h:    1,
			want: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.h.String(), "String()")
		})
	}
}
