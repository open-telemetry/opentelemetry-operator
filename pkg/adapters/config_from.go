package adapters

import (
	"context"
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

var (
	// ErrNoInstance represents an error when an instance isn't available in the context
	ErrNoInstance = errors.New("couldn't get the opentelemetry-collector instance from the context")

	// ErrInvalidYAML represents an error in the format of the configuration file
	ErrInvalidYAML = errors.New("couldn't parse the opentelemetry-collector configuration to assemble a list of ports for the service")
)

// ConfigFromCtx extracts a configuration map from the given context.
// If an instance isn't available in the context, ErrNoInstance is returned.
// If the .Spec.Config isn't a valid YAML, ErrInvalidYAML is returned.
func ConfigFromCtx(ctx context.Context) (map[interface{}]interface{}, error) {
	switch instance := ctx.Value(opentelemetry.ContextInstance).(type) {
	case *v1alpha1.OpenTelemetryCollector:
		return ConfigFromString(instance.Spec.Config)
	default:
		return nil, ErrNoInstance
	}
}

// ConfigFromString extracts a configuration map from the given string.
// If the given string isn't a valid YAML, ErrInvalidYAML is returned.
func ConfigFromString(configStr string) (map[interface{}]interface{}, error) {
	config := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, ErrInvalidYAML
	}

	return config, nil
}
