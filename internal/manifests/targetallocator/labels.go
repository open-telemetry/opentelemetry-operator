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
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// Labels return the common labels to all TargetAllocator objects that are part of a managed OpenTelemetryCollector.
func Labels(instance v1alpha2.OpenTelemetryCollector, name string) map[string]string {
	return manifestutils.Labels(instance.ObjectMeta, name, instance.Spec.TargetAllocator.Image, ComponentOpenTelemetryTargetAllocator, nil)
}

// SelectorLabels return the selector labels for Target Allocator Pods.
func SelectorLabels(instance v1alpha2.OpenTelemetryCollector) map[string]string {
	selectorLabels := manifestutils.SelectorLabels(instance.ObjectMeta, ComponentOpenTelemetryTargetAllocator)
	// TargetAllocator uses the name label as well for selection
	// This is inconsistent with the Collector, but changing is a somewhat painful breaking change
	selectorLabels["app.kubernetes.io/name"] = naming.TargetAllocator(instance.Name)
	return selectorLabels
}
