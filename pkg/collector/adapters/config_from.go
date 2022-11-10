// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package adapters is for data conversion.
package adapters

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrInvalidYAML represents an error in the format of the configuration file.
	ErrInvalidYAML = errors.New("couldn't parse the opentelemetry-collector configuration")
)

// ConfigFromString extracts a configuration map from the given string.
// If the given string isn't a valid YAML, ErrInvalidYAML is returned.
func ConfigFromString(configStr string) (map[interface{}]interface{}, error) {
	config := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, ErrInvalidYAML
	}

	return config, nil
}

// GetConfigString returns the string value for the Collector, whether that is from spec.config as a string literal
// or from spec.configMap as stored in a ConfigMap. The value can be used later in ConfigFromString.
func GetConfigString(client client.Client, logger logr.Logger, configStr string, configMap v1.ConfigMapKeySelector, namespace string) string {
	// the webhook validates that only one of spec.Config or spec.ConfigMap will be set, so we can assume that here
	if len(configStr) > 0 {
		return configStr
	}

	obj := &v1.ConfigMap{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: configMap.Name}, obj)
	if err != nil {
		logger.Error(err, "error getting configmap")
		return ""
	}
	return obj.Data[configMap.Key]
}
