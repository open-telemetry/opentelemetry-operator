// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	go_yaml "github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFiles(t *testing.T) {
	files, err := os.ReadDir("./testdata")
	require.NoError(t, err)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "otelcol-") {
			continue
		}

		testFile := path.Join("./testdata", file.Name())
		t.Run(testFile, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(testFile)
			require.NoError(t, err)

			collectorJson, err := go_yaml.YAMLToJSON(collectorYaml)
			require.NoError(t, err)

			cfg := &Config{}
			err = json.Unmarshal(collectorJson, cfg)
			require.NoError(t, err)
			jsonCfg, err := json.Marshal(cfg)
			require.NoError(t, err)

			assert.JSONEq(t, string(collectorJson), string(jsonCfg))
			yamlCfg, err := go_yaml.JSONToYAML(jsonCfg)
			require.NoError(t, err)
			assert.YAMLEq(t, string(collectorYaml), string(yamlCfg))
		})
	}
}

func TestConfigFiles_go_yaml(t *testing.T) {
	files, err := os.ReadDir("./testdata")
	require.NoError(t, err)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "otelcol-") {
			continue
		}

		testFile := path.Join("./testdata", file.Name())
		t.Run(testFile, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(testFile)
			require.NoError(t, err)

			cfg := &Config{}
			err = go_yaml.Unmarshal(collectorYaml, cfg)
			require.NoError(t, err)
			yamlCfg, err := go_yaml.Marshal(cfg)
			require.NoError(t, err)

			require.NoError(t, err)
			assert.YAMLEq(t, string(collectorYaml), string(yamlCfg))
		})
	}
}

func TestAnyConfigDeepCopyInto_NestedMapIndependence(t *testing.T) {
	src := AnyConfig{Object: map[string]any{
		"prometheus": map[string]any{
			"config": map[string]any{
				"scrape_configs": []any{
					map[string]any{
						"job_name": "kubelet",
						"tls_config": map[string]any{
							"ca_file":              "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
							"insecure_skip_verify": true,
						},
					},
				},
			},
		},
	}}

	dst := src.DeepCopy()

	// Mutate a nested map in the copy (simulates TLS profile injection).
	scrapeConfigs := dst.Object["prometheus"].(map[string]any)["config"].(map[string]any)["scrape_configs"].([]any)
	tlsConfig := scrapeConfigs[0].(map[string]any)["tls_config"].(map[string]any)
	tlsConfig["min_version"] = "TLS12"

	// Source nested map must be unaffected.
	srcTLS := src.Object["prometheus"].(map[string]any)["config"].(map[string]any)["scrape_configs"].([]any)[0].(map[string]any)["tls_config"].(map[string]any)
	assert.NotContains(t, srcTLS, "min_version", "DeepCopy must produce independent nested maps; source was mutated through the copy")
}

func TestAnyConfigDeepCopyInto_NilObject(t *testing.T) {
	src := AnyConfig{Object: nil}
	dst := src.DeepCopy()
	assert.Nil(t, dst.Object)
}

func TestAnyConfigDeepCopyInto_EmptyObject(t *testing.T) {
	src := AnyConfig{Object: map[string]any{}}
	dst := src.DeepCopy()
	assert.NotNil(t, dst.Object)
	assert.Empty(t, dst.Object)
	// Mutating dst should not affect src.
	dst.Object["key"] = "value"
	assert.Empty(t, src.Object)
}

func TestAnyConfigDeepCopyInto_PreservesValues(t *testing.T) {
	src := AnyConfig{Object: map[string]any{
		"string_val": "hello",
		"number_val": float64(42),
		"bool_val":   true,
		"nested": map[string]any{
			"inner": "value",
			"list":  []any{"a", "b"},
		},
	}}

	dst := src.DeepCopy()

	assert.Equal(t, "hello", dst.Object["string_val"])
	assert.Equal(t, float64(42), dst.Object["number_val"])
	assert.Equal(t, true, dst.Object["bool_val"])
	nested := dst.Object["nested"].(map[string]any)
	assert.Equal(t, "value", nested["inner"])
	assert.Equal(t, []any{"a", "b"}, nested["list"])
}
