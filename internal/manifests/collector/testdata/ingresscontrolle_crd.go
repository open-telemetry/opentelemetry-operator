package testdata

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenShiftIngressControllerCRD as go structure.
var OpenShiftIngressControllerCRD = &apiextensionsv1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ingresscontrollers.operator.openshift.io",
	},
	Spec: apiextensionsv1.CustomResourceDefinitionSpec{
		Group: "operator.openshift.io",
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
				AdditionalPrinterColumns: []apiextensionsv1.CustomResourceColumnDefinition{},
				Subresources: &apiextensionsv1.CustomResourceSubresources{
					Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
				},
			},
		},
		Scope: apiextensionsv1.NamespaceScoped,
		Names: apiextensionsv1.CustomResourceDefinitionNames{
			Plural:   "ingresscontrollers",
			Singular: "ingresscontroller",
			Kind:     "IngressController",
		},
	},
}
