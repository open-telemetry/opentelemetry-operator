// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFipsCheck(t *testing.T) {
	fipsCheck := NewFipsCheck([]string{"rec1", "rec2"}, []string{"exp1"}, []string{"processor"}, []string{"ext1"})
	blocked := fipsCheck.DisabledComponents(
		map[string]interface{}{"otlp": true, "rec1/my": true},
		map[string]interface{}{"exp1": true},
		map[string]interface{}{"processor": true},
		map[string]interface{}{"ext1": true})

	assert.Equal(t, []string{"rec1", "exp1", "processor", "ext1"}, blocked)
}
