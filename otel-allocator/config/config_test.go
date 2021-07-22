package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testFile = "./testdata/config_test.yaml"

func TestConfigLoad(t *testing.T) {
	expectedFileSDConfig := map[interface{}]interface{}{
		"files": []interface{}{"./file_sd_test.json"},
	}
	expectedStaticSDConfig := map[interface{}]interface{}{
		"targets": []interface{}{
			"prom.domain:9001",
			"prom.domain:9002",
			"prom.domain:9003",
		},
		"labels": map[interface{}]interface{}{
			"my": "label",
		},
	}

	cfg := Config{}
	err := unmarshall(&cfg, testFile)
	assert.NoError(t, err)

	actualFileSDConfig := cfg.Config.ScrapeConfigs[0]["file_sd_configs"].([]interface{})[0]
	actulaStaticSDConfig := cfg.Config.ScrapeConfigs[0]["static_configs"].([]interface{})[0]

	assert.Equal(t, cfg.LabelSelector["app.kubernetes.io/instance"], "default.test")
	assert.Equal(t, cfg.LabelSelector["app.kubernetes.io/managed-by"], "opentelemetry-operator")
	assert.Equal(t, cfg.Config.ScrapeConfigs[0]["job_name"], "prometheus")
	assert.Equal(t, expectedFileSDConfig, actualFileSDConfig)
	assert.Equal(t, expectedStaticSDConfig, actulaStaticSDConfig)
}
