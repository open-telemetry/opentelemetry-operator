// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	tests := []struct {
		testName     string
		receiverName string
		port         int32
		expected     string
	}{
		{
			testName:     "too_long",
			receiverName: "otlphttpotlphttpotlphttpotlphttpotlphttpotlphttpotlphttpotlphttpotlphttpotlphttpotlphttpotlphttp",
			port:         4318,
			expected:     "port-4318",
		},
		{
			testName:     "with underscore",
			receiverName: "otlp_http",
			port:         4318,
			expected:     "otlp-http",
		},
		{
			testName:     "with slash",
			receiverName: "otlp/http",
			port:         4318,
			expected:     "otlp-http",
		},
		{
			testName:     "not DNS",
			receiverName: "otlp&&**http",
			port:         4318,
			expected:     "port-4318",
		},
		{
			testName:     "ok",
			receiverName: "otlphttp",
			port:         4318,
			expected:     "otlphttp",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			name := PortName(test.receiverName, test.port)
			assert.Equal(t, test.expected, name)
		})
	}
}
