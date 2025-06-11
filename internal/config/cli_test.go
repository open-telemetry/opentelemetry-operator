// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyFlag(t *testing.T) {
	args = []string{"--enable-go-instrumentation=true"}
	c := New()
	require.False(t, c.EnableGoAutoInstrumentation)
	require.NoError(t, ApplyCLI(&c))
	require.True(t, c.EnableGoAutoInstrumentation)
}
