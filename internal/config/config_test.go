// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
)

func TestToStringMap(t *testing.T) {
	cfg := Config{
		OpenShiftRoutesAvailability:       openshift.RoutesAvailable,
		OperatorOpAMPBridgeConfigMapEntry: "foo.yaml",
		CollectorImage:                    "myexample:1.0",
		IgnoreMissingCollectorCRDs:        true,
	}
	assert.Equal(t, map[string]string{
		"auto-instrumentation-apache-httpd-image": "",
		"auto-instrumentation-dot-net-image":      "",
		"auto-instrumentation-go-image":           "",
		"auto-instrumentation-java-image":         "",
		"auto-instrumentation-nginx-image":        "",
		"auto-instrumentation-node-js-image":      "",
		"auto-instrumentation-python-image":       "",
		"cert-manager-availability":               "0",
		"collector-availability":                  "0",
		"collector-configmap-entry":               "",
		"collector-image":                         "myexample:1.0",
		"create-rbac-permissions":                 "0",
		"create-service-monitor-operator-metrics": "false",
		"enable-apache-httpd-instrumentation":     "false",
		"enable-cr-metrics":                       "false",
		"enable-dot-net-auto-instrumentation":     "false",
		"enable-go-auto-instrumentation":          "false",
		"enable-java-auto-instrumentation":        "false",
		"enable-leader-election":                  "false",
		"enable-multi-instrumentation":            "false",
		"enable-nginx-auto-instrumentation":       "false",
		"enable-node-js-auto-instrumentation":     "false",
		"enable-python-auto-instrumentation":      "false",
		"feature-gates":                           "",
		"fips-disabled-components":                "",
		"ignore-missing-collector-crds":           "true",
		"metrics-addr":                            "",
		"metrics-secure":                          "false",
		"metrics-tls-cert-file":                   "",
		"metrics-tls-key-file":                    "",
		"opampbridge-availability":                "0",
		"open-shift-routes-availability":          "0",
		"openshift-create-dashboard":              "false",
		"operator-op-amp-bridge-configmap-entry":  "foo.yaml",
		"operatoropampbridge-image":               "",
		"pprof-addr":                              "",
		"health-probe-addr":                       "",
		"prometheus-cr-availability":              "0",
		"target-allocator-availability":           "0",
		"target-allocator-configmap-entry":        "",
		"targetallocator-image":                   "",
		"webhook-port":                            "0",
		"enable-webhooks":                         "false",
	}, cfg.ToStringMap())
}

func TestMissingFile(t *testing.T) {
	err := ApplyConfigFile("testdata/missing.yaml", &Config{})
	require.Error(t, err, "missing file")
}

func TestApply(t *testing.T) {
	tests := []struct {
		description string
		configFile  string
		args        []string
		config      Config
		err         string
	}{
		{
			description: "default",
			config:      New(),
			err:         "",
		},
		{
			description: "missing file",
			configFile:  "missing.yaml",
			err:         "failed to apply file config: open missing.yaml: no such file or directory",
		},
		{
			description: "bad CLI args",
			args:        []string{"--webhook-port=1foo"},
			err:         `failed to apply cli config: invalid argument "1foo" for "--webhook-port" flag: strconv.ParseInt: parsing "1foo": invalid syntax`,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			if len(test.args) > 0 {
				oldArgs := args
				args = test.args
				t.Cleanup(func() {
					args = oldArgs
				})
			}
			err := test.config.Apply(test.configFile)
			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.err)
			}
		})
	}
}
