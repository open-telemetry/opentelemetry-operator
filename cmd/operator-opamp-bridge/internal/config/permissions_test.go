// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestOperatorPermissionsPatchOnlyWithRestartCommand(t *testing.T) {
	countPatch := func(restartEnabled bool) int {
		cfg := NewConfig(logr.Discard())
		cfg.Capabilities = map[Capability]bool{AcceptsRestartCommand: restartEnabled}
		perms, err := cfg.OperatorPermissions()
		require.NoError(t, err)
		n := 0
		for _, p := range perms {
			if p.Verb == "patch" {
				n++
			}
		}
		return n
	}
	require.Equal(t, 0, countPatch(false))
	require.Equal(t, 3, countPatch(true))
}
