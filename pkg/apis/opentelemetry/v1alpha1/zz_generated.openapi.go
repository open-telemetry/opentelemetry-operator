// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollector":       schema_pkg_apis_opentelemetry_v1alpha1_OpenTelemetryCollector(ref),
		"./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollectorSpec":   schema_pkg_apis_opentelemetry_v1alpha1_OpenTelemetryCollectorSpec(ref),
		"./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollectorStatus": schema_pkg_apis_opentelemetry_v1alpha1_OpenTelemetryCollectorStatus(ref),
	}
}

func schema_pkg_apis_opentelemetry_v1alpha1_OpenTelemetryCollector(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "OpenTelemetryCollector is the Schema for the opentelemetrycollectors API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollectorSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollectorStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollectorSpec", "./pkg/apis/opentelemetry/v1alpha1.OpenTelemetryCollectorStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_opentelemetry_v1alpha1_OpenTelemetryCollectorSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"config": {
						SchemaProps: spec.SchemaProps{
							Description: "Config is the raw JSON to be used as the collector's configuration. Refer to the OpenTelemetry Collector documentation for details.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"args": {
						SchemaProps: spec.SchemaProps{
							Description: "Args is the set of arguments to pass to the OpenTelemetry Collector binary",
							Type:        []string{"object"},
							AdditionalProperties: &spec.SchemaOrBool{
								Allows: true,
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"replicas": {
						SchemaProps: spec.SchemaProps{
							Description: "Replicas is the number of pod instances for the underlying OpenTelemetry Collector",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"image": {
						SchemaProps: spec.SchemaProps{
							Description: "Image indicates the container image to use for the OpenTelemetry Collector.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"mode": {
						SchemaProps: spec.SchemaProps{
							Description: "Mode represents how the collector should be deployed (deployment vs. daemonset)",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"serviceAccount": {
						SchemaProps: spec.SchemaProps{
							Description: "ServiceAccount indicates the name of an existing service account to use with this instance.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"volumeMounts": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "atomic",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "VolumeMounts represents the mount points to use in the underlying collector deployment(s)",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("k8s.io/api/core/v1.VolumeMount"),
									},
								},
							},
						},
					},
					"volumes": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "atomic",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "Volumes represents which volumes to use in the underlying collector deployment(s).",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("k8s.io/api/core/v1.Volume"),
									},
								},
							},
						},
					},
					"ports": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "atomic",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "Ports allows a set of ports to be exposed by the underlying v1.Service. By default, the operator will attempt to infer the required ports by parsing the .Spec.Config property but this property can be used to open aditional ports that can't be inferred by the operator, like for custom receivers.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("k8s.io/api/core/v1.ServicePort"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/api/core/v1.ServicePort", "k8s.io/api/core/v1.Volume", "k8s.io/api/core/v1.VolumeMount"},
	}
}

func schema_pkg_apis_opentelemetry_v1alpha1_OpenTelemetryCollectorStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"replicas": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"version": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
				Required: []string{"replicas", "version"},
			},
		},
	}
}
