package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	for _, tt := range []struct {
		format   string
		max      int
		values   []interface{}
		expected string
		cap      string
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
			format:   "%d-%s-collector",
			max:      63,
			values:   []interface{}{42, "d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96-11ea-b174-c85b7644b6b5"},
			expected: "42-d0c1e62-4d96-11ea-b174-c85b7644b6b5-5d0c1e62-4d96--collector",
			cap:      "first value gets passed, second truncated",
		},
	} {
		assert.Equal(t, tt.expected, Truncate(tt.format, tt.max, tt.values...))
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