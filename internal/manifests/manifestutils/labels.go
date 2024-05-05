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

package manifestutils

import (
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func IsFilteredSet(sourceSet string, filterSet []string) bool {
	for _, pattern := range filterSet {
		if match, _ := regexp.MatchString(pattern, sourceSet); match {
			return match
		}
	}
	return false
}

// Labels return the common labels to all objects that are part of a managed CR.
func Labels(instance metav1.ObjectMeta, name string, image string, component string, filterLabels []string) map[string]string {
	var versionLabel string
	// new map every time, so that we don't touch the instance's label
	base := map[string]string{}
	if nil != instance.Labels {
		for k, v := range instance.Labels {
			if !IsFilteredSet(k, filterLabels) {
				base[k] = v
			}
		}
	}

	for k, v := range SelectorLabels(instance, component) {
		base[k] = v
	}

	version := strings.Split(image, ":")
	for _, v := range version {
		if strings.HasSuffix(v, "@sha256") {
			versionLabel = strings.TrimSuffix(v, "@sha256")
		}
	}
	switch lenVersion := len(version); lenVersion {
	case 3:
		base["app.kubernetes.io/version"] = versionLabel
	case 2:
		base["app.kubernetes.io/version"] = naming.Truncate("%s", 63, version[len(version)-1])
	default:
		base["app.kubernetes.io/version"] = "latest"
	}

	// Don't override the app name if it already exists
	if _, ok := base["app.kubernetes.io/name"]; !ok {
		base["app.kubernetes.io/name"] = name
	}
	return base
}

// SelectorLabels return the common labels to all objects that are part of a managed CR to use as selector.
// Selector labels are immutable for Deployment, StatefulSet and DaemonSet, therefore, no labels in selector should be
// expected to be modified for the lifetime of the object.
func SelectorLabels(instance metav1.ObjectMeta, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   naming.Truncate("%s.%s", 63, instance.Namespace, instance.Name),
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  component,
	}
}

// SelectorLabels return the selector labels for Target Allocator Pods.
func TASelectorLabels(instance v1beta1.TargetAllocator, component string) map[string]string {
	selectorLabels := SelectorLabels(instance.ObjectMeta, component)

	// TargetAllocator uses the name label as well for selection
	// This is inconsistent with the Collector, but changing is a somewhat painful breaking change
	// Don't override the app name if it already exists
	if name, ok := instance.ObjectMeta.Labels["app.kubernetes.io/name"]; ok {
		selectorLabels["app.kubernetes.io/name"] = name
	} else {
		selectorLabels["app.kubernetes.io/name"] = naming.TargetAllocator(instance.Name)
	}
	return selectorLabels
}
