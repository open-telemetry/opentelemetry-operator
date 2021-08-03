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

package reconcile

import (
	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

func IsAllocatorEnabled(params Params) bool {
	return params.Instance.Spec.Mode == v1alpha1.ModeStatefulSet && params.Instance.Spec.TargetAllocator.Enabled
}

func GetPromConfig(params Params) (map[interface{}]interface{}, error) {
	config, err := adapters.ConfigFromString(params.Instance.Spec.Config)
	if err != nil {
		return nil, err
	}

	return ta.ConfigToPromConfig(config)
}
