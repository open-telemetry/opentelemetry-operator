// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"

	"dario.cat/mergo"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// please see: https://github.com/open-telemetry/opentelemetry-operator/issues/2942
func MergeDefaults(base, override *v1beta1.Config) (*v1beta1.Config, error) {
	baseMap, err := ConfigFromStruct(base)
	if err != nil {
		return nil, err
	}
	overrideMap, err := ConfigFromStruct(override)
	if err != nil {
		return nil, err
	}
	if err := mergo.Merge(&baseMap, overrideMap, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merge collector config defaults: %w", err)
	}
	return ConfigToStruct(baseMap)
}
