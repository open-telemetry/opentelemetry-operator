// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenShiftRouteCRD as go structure.
var OpenShiftRouteCRD = &apiextensionsv1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "routes.route.openshift.io",
	},
	Spec: apiextensionsv1.CustomResourceDefinitionSpec{
		Group: "route.openshift.io",
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
						Name:     "Host",
						Type:     "string",
						JSONPath: ".status.ingress[0].host",
					},
					{
						Name:     "Admitted",
						Type:     "string",
						JSONPath: `.status.ingress[0].conditions[?(@.type=="Admitted")].status`,
					},
					{
						Name:     "Service",
						Type:     "string",
						JSONPath: ".spec.to.name",
					},
					{
						Name:     "TLS",
						Type:     "string",
						JSONPath: ".spec.tls.type",
					},
				},
				Subresources: &apiextensionsv1.CustomResourceSubresources{
					Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: apiextensionsv1.NamespaceScoped,
		Names: apiextensionsv1.CustomResourceDefinitionNames{
			Plural:   "routes",
			Singular: "route",
			Kind:     "Route",
		},
	},
}
