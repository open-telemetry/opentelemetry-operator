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

func TestFilterFlags(t *testing.T) {
	oldArgs := args
	args = []string{
		"--labels-filter=.*filter.out",
		"--annotations-filter=another.*.filter",
	}
	t.Cleanup(func() {
		args = oldArgs
	})
	c := New()
	require.NoError(t, ApplyCLI(&c))
	require.Equal(t, []string{".*filter.out"}, c.LabelsFilter)
	require.Equal(t, []string{"another.*.filter"}, c.AnnotationsFilter)
}
