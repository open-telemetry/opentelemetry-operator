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
