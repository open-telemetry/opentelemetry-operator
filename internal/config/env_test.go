// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An invalid value must not silently turn an option off or override the
// config file, so ApplyEnvVars reports it and keeps the current value.
func TestApplyEnvVarsInvalidValues(t *testing.T) {
	t.Setenv("ENABLE_WEBHOOKS", "false")
	t.Setenv("WEBHOOK_PORT", "9444")
	t.Setenv("ENABLE_LEADER_ELECTION", "enabled")
	t.Setenv("OPENSHIFT_WEBHOOK_REPLICAS", "two")
	t.Setenv("TLS_CURVE_PREFERENCES", "29,x25519")

	cfg := New()
	cfg.EnableLeaderElection = true
	err := ApplyEnvVars(&cfg)

	require.EqualError(t, err,
		`invalid value "enabled" for ENABLE_LEADER_ELECTION: strconv.ParseBool: parsing "enabled": invalid syntax`+"\n"+
			`invalid value "29,x25519" for TLS_CURVE_PREFERENCES: strconv.ParseInt: parsing "x25519": invalid syntax`+"\n"+
			`invalid value "two" for OPENSHIFT_WEBHOOK_REPLICAS: strconv.ParseInt: parsing "two": invalid syntax`)
	assert.False(t, cfg.EnableWebhooks)
	assert.Equal(t, 9444, cfg.WebhookPort)
	assert.True(t, cfg.EnableLeaderElection)
	assert.Equal(t, int32(defaultOpenShiftWebhookReplicas), cfg.OpenShiftWebhookReplicas)
	assert.Nil(t, cfg.TLS.CurvePreferences)
}
