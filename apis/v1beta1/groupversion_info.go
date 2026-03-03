// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package v1beta1 contains API Schema definitions for the  v1beta1 API group
// +kubebuilder:object:generate=true
// +groupName=opentelemetry.io
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "opentelemetry.io", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes registers the API types with the given scheme.
func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion,
		&OpenTelemetryCollector{},
		&OpenTelemetryCollectorList{},
	)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
