// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HTTPRouteCRD as go structure.
var HTTPRouteCRD = &apiextensionsv1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "httproutes.gateway.networking.k8s.io",
		Annotations: map[string]string{
			"api-approved.kubernetes.io": "https://github.com/kubernetes-sigs/gateway-api",
		},
	},
	Spec: apiextensionsv1.CustomResourceDefinitionSpec{
		Group: "gateway.networking.k8s.io",
		Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
			{
				Name:    "v1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type:                   "object",
						XPreserveUnknownFields: func(v bool) *bool { return &v }(true),
					},
				},
				AdditionalPrinterColumns: []apiextensionsv1.CustomResourceColumnDefinition{
					{
						Name:     "Hostnames",
						Type:     "string",
						JSONPath: ".spec.hostnames",
					},
					{
						Name:     "Age",
						Type:     "date",
						JSONPath: ".metadata.creationTimestamp",
					},
				},
				Subresources: &apiextensionsv1.CustomResourceSubresources{
					Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: apiextensionsv1.NamespaceScoped,
		Names: apiextensionsv1.CustomResourceDefinitionNames{
			Plural:     "httproutes",
			Singular:   "httproute",
			Kind:       "HTTPRoute",
			ShortNames: []string{"httproute"},
			Categories: []string{"gateway-api"},
		},
	},
}
