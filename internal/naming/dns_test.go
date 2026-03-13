// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Additional copyrights:
// Copyright The Jaeger Authors

package naming

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDnsName(t *testing.T) {
	tests := []struct {
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
	rule := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	for _, tt := range tests {
		assert.Equal(t, tt.out, DNSName(tt.in))
		matched := rule.MatchString(tt.out)
		assert.True(t, matched, "%v is not a valid name", tt.out)
	}
}
