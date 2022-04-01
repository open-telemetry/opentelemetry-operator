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

package collector

import (
	"strings"
	"regexp"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

func isFilteredLabel(label string, filterLabels []string) bool {
	for _, pattern := range filterLabels {
		match, _ := regexp.MatchString(pattern, label)
		return match
	}

	return false
}

// Labels return the common labels to all objects that are part of a managed OpenTelemetryCollector.
func Labels(instance v1alpha1.OpenTelemetryCollector, filterLabels []string) map[string]string {
	// new map every time, so that we don't touch the instance's label
	base := map[string]string{}
	if nil != instance.Labels {
		for k, v := range instance.Labels {
			if !isFilteredLabel(k, filterLabels) {
				base[k] = v
			}
		}
	}

	base["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	base["app.kubernetes.io/instance"] = naming.Truncate("%s.%s", 63, instance.Namespace, instance.Name)
	base["app.kubernetes.io/part-of"] = "opentelemetry"
	base["app.kubernetes.io/component"] = "opentelemetry-collector"
	version := strings.Split(instance.Spec.Image, ":")
	if len(version) > 1 {
		base["app.kubernetes.io/version"] = version[len(version)-1]
	} else {
		base["app.kubernetes.io/version"] = "latest"
	}

	return base
}
