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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDnsName(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"simplest", "simplest"},
		{"instance.with.dots-collector-headless", "instance-with-dots-collector-headless"},
		{"TestQueryDottedServiceName.With.Dots", "testquerydottedservicename-with-dots"},
		{"Service🦄", "servicea"},
		{"📈Stock-Tracker", "astock-tracker"},
		{"-📈Stock-Tracker", "a-stock-tracker"},
		{"📈", "a"},
		{"foo-", "fooa"},
		{"-foo", "afoo"},
	}
	rule, err := regexp.Compile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	assert.NoError(t, err)

	for _, tt := range tests {
		assert.Equal(t, tt.out, DNSName(tt.in))
		matched := rule.Match([]byte(tt.out))
		assert.True(t, matched, "%v is not a valid name", tt.out)
	}
}
