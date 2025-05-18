// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"regexp"
	"strings"

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

type labelConfig struct {
	filterLabels     []string
	additionalLabels map[string]string
}

type LabelOption func(c *labelConfig)

func WithFilterLabels(filters []string) LabelOption {
	return func(c *labelConfig) {
		c.filterLabels = filters
	}
}

func WithAdditionalLabels(labels map[string]string) LabelOption {
	return func(c *labelConfig) {
		c.additionalLabels = labels
	}
}

// Labels return the common labels to all objects that are part of a managed CR.
func Labels(instance metav1.ObjectMeta, name string, image string, component string, opts ...LabelOption) map[string]string {
	var config labelConfig
	for _, opt := range opts {
		opt(&config)
	}

	// new map every time, so that we don't touch the instance's label
	base := map[string]string{}
	for k, v := range config.additionalLabels {
		base[k] = v
	}
	if instance.Labels != nil {
		for k, v := range instance.Labels {
			if !IsFilteredSet(k, config.filterLabels) {
				base[k] = v
			}
		}
	}

	for k, v := range SelectorLabels(instance, component) {
		base[k] = v
	}

	version := strings.Split(image, ":")
	var versionLabel string
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
func TASelectorLabels(instance v1alpha1.TargetAllocator, component string) map[string]string {
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
