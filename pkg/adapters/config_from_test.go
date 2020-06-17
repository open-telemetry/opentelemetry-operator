package adapters

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestConfigFromCtx(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{}
	instance.Spec.Config = "mykey: myval"
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)

	// test
	config, err := ConfigFromCtx(ctx)

	// verify
	assert.NoError(t, err)
	assert.EqualValues(t, "myval", config["mykey"])
}

func TestConfigFromCtxNoContext(t *testing.T) {
	// test
	config, err := ConfigFromCtx(context.Background())

	// verify
	assert.Equal(t, ErrNoInstance, err)
	assert.Nil(t, config)
}

func TestConfigFromStringInvalidYAML(t *testing.T) {
	// test
	config, err := ConfigFromString("🦄")

	// verify
	assert.Equal(t, ErrInvalidYAML, err)
	assert.Nil(t, config)
}

func TestConfigFromStringEmptyMap(t *testing.T) {
	// test
	config, err := ConfigFromString("")

	// verify
	assert.Nil(t, err)
	assert.Len(t, config, 0)
}
