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

package targetallocator

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Labels return the common labels to all TargetAllocator objects that are part of a managed OpenTelemetryCollector.
func Labels(instance v1alpha1.OpenTelemetryCollector, name string) map[string]string {
	// new map every time, so that we don't touch the instance's label
	base := map[string]string{}
	if nil != instance.Labels {
		for k, v := range instance.Labels {
			base[k] = v
		}
	}

	base["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	base["app.kubernetes.io/instance"] = naming.Truncate("%s.%s", 63, instance.Namespace, instance.Name)
	base["app.kubernetes.io/part-of"] = "opentelemetry"
	base["app.kubernetes.io/component"] = "opentelemetry-targetallocator"

	if _, ok := base["app.kubernetes.io/name"]; !ok {
		base["app.kubernetes.io/name"] = name
	}

	return base
}
