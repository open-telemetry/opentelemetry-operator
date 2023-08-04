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

// Additional copyrights:
// Copyright The Jaeger Authors

package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	for _, tt := range []struct {
		format   string
		expected string
		cap      string
		values   []interface{}
		max      int
	}{
		{
			format:   "%s-collector",
			max:      63,
			values:   []interface{}{"simplest"},
			expected: "simplest-collector",
			cap:      "the standard case",
		},
		{
			format:   "d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85b7644b6b5",
			max:      63,
			values:   []interface{}{},
			expected: "d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85",
			cap:      "first N case",
		},
		{
			format:   "%s-collector",
			max:      63,
			values:   []interface{}{"d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85b7644b6b5"},
			expected: "d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11e-collector",
			cap:      "instance + fixed within bounds",
		},
		{
			format:   "%s-%s-collector",
			max:      63,
			values:   []interface{}{"d0c1e62", "4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85b7644b6b5"},
			expected: "4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174--collector",
			cap:      "first value gets dropped, second truncated",
		},
		{
			format:   "%s-%s-collector",
			max:      63,
			values:   []interface{}{"4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85b7644b6b5", "d0c1e62"},
			expected: "4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11e-d0c1e62-collector",
			cap:      "first value gets truncated, second added",
		},
		{
			format:   "%d-%s-collector",
			max:      63,
			values:   []interface{}{42, "d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85b7644b6b5"},
			expected: "42-d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96--collector",
			cap:      "first value gets passed, second truncated",
		},
	} {
		t.Run(tt.cap, func(t *testing.T) {
			assert.Equal(t, tt.expected, Truncate(tt.format, tt.max, tt.values...))
		})
	}
}

func TestTrimNonAlphaNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "-%$#ThisIsALabel",
			expected: "ThisIsALabel",
		},

		{
			input:    "label-invalid--_truncated-.",
			expected: "label-invalid--_truncated",
		},

		{
			input:    "--(label-invalid--_truncated-#.1.",
			expected: "label-invalid--_truncated-#.1",
		},

		{
			input:    "12ValidLabel3",
			expected: "12ValidLabel3",
		},
	}

	for _, test := range tests {
		output := trimNonAlphaNumeric(test.input)
		assert.Equal(t, test.expected, output)
	}
}
