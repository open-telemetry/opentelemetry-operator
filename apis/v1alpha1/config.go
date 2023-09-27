package v1alpha1

import (
	"encoding/json"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
)

func MustParseConfigSpec(in string) *ConfigSpec {
	cfg := &ConfigSpec{}
	yaml.Unmarshal([]byte(in), &cfg)
	return cfg
}

type ConfigSpec struct {
	Exporters  *runtime.RawExtension `json:"exporters,omitempty"`
	Receivers  *runtime.RawExtension `json:"receivers,omitempty"`
	Connectors *runtime.RawExtension `json:"connectors,omitempty"`
	Processors *runtime.RawExtension `json:"processors,omitempty"`
	Service    *runtime.RawExtension `json:"service,omitempty"`
}

func (c *ConfigSpec) String() string {
	v, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(v)
}
