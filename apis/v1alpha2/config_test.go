package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var cfg = `
receivers:
  jaeger:
    protocols:
     thrift_compact:

processors:
  batch:

exporters:
  debug: null

service:
  pipelines:
   traces:
    receivers: [jaeger]
    processors: []
    exporters: [debug]
`

func TestConfigMarshalling(t *testing.T) {
	jsonCfg, err := yaml.YAMLToJSON([]byte(cfg))
	require.NoError(t, err)

	c := &Config{}
	err = c.UnmarshalJSON(jsonCfg)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"batch": nil}, c.cfg["processors"])
	assert.Equal(t, map[string]interface{}{"debug": nil}, c.cfg["exporters"])

	json, err := c.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, jsonCfg, json)
}
