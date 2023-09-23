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
