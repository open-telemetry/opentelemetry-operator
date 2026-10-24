// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"maps"
	"regexp"

	"github.com/distribution/reference"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func IsFilteredSet(sourceSet string, filterSet []string) bool {
	for _, basePattern := range filterSet {
		pattern, _ := regexp.Compile(basePattern)
		if match := pattern.MatchString(sourceSet); match {
			return match
		}
	}
	return false
}

// Labels return the common labels to all objects that are part of a managed CR.
func Labels(instance metav1.ObjectMeta, name, image, component string, filterLabels []string) map[string]string {
	// new map every time, so that we don't touch the instance's label
	base := map[string]string{}
	if instance.Labels != nil {
		for k, v := range instance.Labels {
			if !IsFilteredSet(k, filterLabels) {
				base[k] = v
			}
		}
	}

	maps.Copy(base, SelectorLabels(instance, component))

	base["app.kubernetes.io/version"] = imageVersionLabel(image)

	// Don't override the app name if it already exists
	if _, ok := base["app.kubernetes.io/name"]; !ok {
		base["app.kubernetes.io/name"] = name
	}
	return base
}

// imageVersionLabel returns the value for the app.kubernetes.io/version label:
// the image tag if there is one, otherwise the digest, otherwise "latest".
func imageVersionLabel(image string) string {
	ref, err := reference.Parse(image)
	if err != nil {
		return "latest"
	}
	if tagged, ok := ref.(reference.Tagged); ok {
		return naming.Truncate("%s", 63, tagged.Tag())
	}
	if digested, ok := ref.(reference.Digested); ok {
		return naming.Truncate("%s", 63, digested.Digest().Encoded())
	}
	return "latest"
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
func TASelectorLabels(instance v1alpha1.TargetAllocator, component string) map[string]string {
	selectorLabels := SelectorLabels(instance.ObjectMeta, component)

	// TargetAllocator uses the name label as well for selection
	// This is inconsistent with the Collector, but changing is a somewhat painful breaking change
	// Don't override the app name if it already exists
	if name, ok := instance.Labels["app.kubernetes.io/name"]; ok {
		selectorLabels["app.kubernetes.io/name"] = name
	} else {
		selectorLabels["app.kubernetes.io/name"] = naming.TargetAllocator(instance.Name)
	}
	return selectorLabels
}

// AddExtraLabels adds extra labels to the ServiceMonitor labels map.
func AddExtraLabels(logger *logr.Logger, labels, extraLabels map[string]string) {
	for k, v := range extraLabels {
		if existingValue, exists := labels[k]; exists {
			if existingValue != v {
				logger.Info("Overwriting label", "label", k, "old value", existingValue, "new value", v)
			}
		}
		labels[k] = v
	}
}
