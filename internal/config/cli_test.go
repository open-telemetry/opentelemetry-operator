// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyFlag(t *testing.T) {
	oldArgs := args
	args = []string{"--enable-go-instrumentation=true"}
	t.Cleanup(func() {
		args = oldArgs
	})
	c := New()
	require.False(t, c.EnableGoAutoInstrumentation)
	require.NoError(t, ApplyCLI(&c))
	require.True(t, c.EnableGoAutoInstrumentation)
}
