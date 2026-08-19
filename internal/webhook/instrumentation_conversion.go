// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const v1alpha1FieldsAnnotation = "instrumentation.opentelemetry.io/v1alpha1-fields"

// v1alpha1Fields holds lossy fields that don't map directly to v1beta1.
type v1alpha1Fields struct {
	JavaVolumeSizeLimit        *resource.Quantity `json:"javaVolumeSizeLimit,omitempty"`
	NodeJSVolumeSizeLimit      *resource.Quantity `json:"nodejsVolumeSizeLimit,omitempty"`
	PythonVolumeSizeLimit      *resource.Quantity `json:"pythonVolumeSizeLimit,omitempty"`
	DotNetVolumeSizeLimit      *resource.Quantity `json:"dotnetVolumeSizeLimit,omitempty"`
	GoVolumeSizeLimit          *resource.Quantity `json:"goVolumeSizeLimit,omitempty"`
	ApacheHttpdVolumeSizeLimit *resource.Quantity `json:"apacheHttpdVolumeSizeLimit,omitempty"`
	ApacheHttpdAttrs           []corev1.EnvVar    `json:"apacheHttpdAttrs,omitempty"`
	NginxVolumeSizeLimit       *resource.Quantity `json:"nginxVolumeSizeLimit,omitempty"`
	NginxAttrs                 []corev1.EnvVar    `json:"nginxAttrs,omitempty"`
}

func InstrumentationConvertTo(src *v1alpha1.Instrumentation, dstRaw any) error {
	switch t := dstRaw.(type) {
	case *v1beta1.Instrumentation:
		dst := dstRaw.(*v1beta1.Instrumentation)
		convertedSrc := tov1beta1Instrumentation(*src)
		dst.ObjectMeta = convertedSrc.ObjectMeta
		dst.Spec = convertedSrc.Spec
		dst.Status = convertedSrc.Status
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
	return nil
}

func InstrumentationConvertFrom(dst *v1alpha1.Instrumentation, srcRaw any) error {
	switch t := srcRaw.(type) {
	case *v1beta1.Instrumentation:
		src := srcRaw.(*v1beta1.Instrumentation)
		srcConverted := tov1alpha1Instrumentation(*src)
		dst.ObjectMeta = srcConverted.ObjectMeta
		dst.Spec = srcConverted.Spec
		dst.Status = srcConverted.Status
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
	return nil
}

func tov1beta1Instrumentation(in v1alpha1.Instrumentation) v1beta1.Instrumentation {
	c := in.DeepCopy()

	// Build envConfig
	var envConfig *v1beta1.EnvConfig
	if c.Spec.Endpoint != "" || len(c.Spec.Propagators) > 0 ||
		c.Spec.Type != "" {
		envConfig = &v1beta1.EnvConfig{
			Exporter: v1beta1.Exporter{
				Endpoint: c.Spec.Endpoint,
				TLS:      convertTLSToV1beta1(c.Spec.TLS),
			},
			Propagators: convertPropagatorsToV1beta1(c.Spec.Propagators),
			Sampler: v1beta1.Sampler{
				Type:     v1beta1.SamplerType(c.Spec.Type),
				Argument: c.Spec.Argument,
			},
		}
	}

	// Build resource config
	resource := v1beta1.Resource{
		Attributes: c.Spec.Resource.Attributes,
	}

	if c.Spec.Resource.AddK8sUIDAttributes {
		resource.K8sMetadata = &v1beta1.K8sMetadataConfig{
			IncludeUIDs: true,
		}
	}

	if c.Spec.Defaults.UseLabelsForResourceAttributes {
		enabled := true
		resource.ServiceMetadata = &v1beta1.ServiceMetadataConfig{
			Enabled: &enabled,
		}
	}

	// Collect lossy fields for annotation
	lossyFields := v1alpha1Fields{
		JavaVolumeSizeLimit:        c.Spec.Java.VolumeSizeLimit,
		NodeJSVolumeSizeLimit:      c.Spec.NodeJS.VolumeSizeLimit,
		PythonVolumeSizeLimit:      c.Spec.Python.VolumeSizeLimit,
		DotNetVolumeSizeLimit:      c.Spec.DotNet.VolumeSizeLimit,
		GoVolumeSizeLimit:          c.Spec.Go.VolumeSizeLimit,
		ApacheHttpdVolumeSizeLimit: c.Spec.ApacheHttpd.VolumeSizeLimit,
		ApacheHttpdAttrs:           c.Spec.ApacheHttpd.Attrs,
		NginxVolumeSizeLimit:       c.Spec.Nginx.VolumeSizeLimit,
		NginxAttrs:                 c.Spec.Nginx.Attrs,
	}

	// Only add annotation if there are lossy fields
	if hasLossyFields(lossyFields) {
		if c.Annotations == nil {
			c.Annotations = make(map[string]string)
		}
		annotationBytes, err := json.Marshal(lossyFields)
		if err == nil {
			c.Annotations[v1alpha1FieldsAnnotation] = string(annotationBytes)
		}
	}

	return v1beta1.Instrumentation{
		ObjectMeta: c.ObjectMeta,
		Status: v1beta1.InstrumentationStatus{
			UpgradeBlockedVersions: c.Status.UpgradeBlockedVersions,
		},
		Spec: v1beta1.InstrumentationSpec{
			EnvConfig: envConfig,
			Resource:  resource,
			Env:       c.Spec.Env,
			Java: v1beta1.Java{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Java.Image,
					VolumeClaimTemplate: c.Spec.Java.VolumeClaimTemplate,
					Env:                 c.Spec.Java.Env,
					Resources:           c.Spec.Java.Resources,
				},
				InitContainer: convertInitContainerToV1beta1(c.Spec.Java.InitContainer),
				Extensions:    convertExtensionsToV1beta1(c.Spec.Java.Extensions),
			},
			NodeJS: v1beta1.NodeJS{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.NodeJS.Image,
					VolumeClaimTemplate: c.Spec.NodeJS.VolumeClaimTemplate,
					Env:                 c.Spec.NodeJS.Env,
					Resources:           c.Spec.NodeJS.Resources,
				},
				InitContainer: convertInitContainerToV1beta1(c.Spec.NodeJS.InitContainer),
			},
			Python: v1beta1.Python{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Python.Image,
					VolumeClaimTemplate: c.Spec.Python.VolumeClaimTemplate,
					Env:                 c.Spec.Python.Env,
					Resources:           c.Spec.Python.Resources,
				},
				InitContainer: convertInitContainerToV1beta1(c.Spec.Python.InitContainer),
			},
			DotNet: v1beta1.DotNet{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.DotNet.Image,
					VolumeClaimTemplate: c.Spec.DotNet.VolumeClaimTemplate,
					Env:                 c.Spec.DotNet.Env,
					Resources:           c.Spec.DotNet.Resources,
				},
				InitContainer: convertInitContainerToV1beta1(c.Spec.DotNet.InitContainer),
			},
			Go: v1beta1.Go{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Go.Image,
					VolumeClaimTemplate: c.Spec.Go.VolumeClaimTemplate,
					Env:                 c.Spec.Go.Env,
					Resources:           c.Spec.Go.Resources,
				},
				SecurityContext: c.Spec.Go.SecurityContext,
			},
			ApacheHttpd: v1beta1.ApacheHttpd{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.ApacheHttpd.Image,
					VolumeClaimTemplate: c.Spec.ApacheHttpd.VolumeClaimTemplate,
					Env:                 c.Spec.ApacheHttpd.Env,
					Resources:           c.Spec.ApacheHttpd.Resources,
				},
				Version:    c.Spec.ApacheHttpd.Version,
				ConfigPath: c.Spec.ApacheHttpd.ConfigPath,
			},
			Nginx: v1beta1.Nginx{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Nginx.Image,
					VolumeClaimTemplate: c.Spec.Nginx.VolumeClaimTemplate,
					Env:                 c.Spec.Nginx.Env,
					Resources:           c.Spec.Nginx.Resources,
				},
				ConfigFile: c.Spec.Nginx.ConfigFile,
			},
			ImagePullPolicy:              c.Spec.ImagePullPolicy,
			InitContainerSecurityContext: c.Spec.InitContainerSecurityContext,
		},
	}
}

func tov1alpha1Instrumentation(in v1beta1.Instrumentation) v1alpha1.Instrumentation {
	c := in.DeepCopy()

	// Extract exporter and sampler from envConfig
	exporter := v1alpha1.Exporter{}
	sampler := v1alpha1.Sampler{}
	var propagators []v1alpha1.Propagator

	if c.Spec.EnvConfig != nil {
		exporter = v1alpha1.Exporter{
			Endpoint: c.Spec.EnvConfig.Exporter.Endpoint,
			TLS:      convertTLSToV1alpha1(c.Spec.EnvConfig.Exporter.TLS),
		}
		sampler = v1alpha1.Sampler{
			Type:     v1alpha1.SamplerType(c.Spec.EnvConfig.Sampler.Type),
			Argument: c.Spec.EnvConfig.Sampler.Argument,
		}
		propagators = convertPropagatorsToV1alpha1(c.Spec.EnvConfig.Propagators)
	}

	// Extract resource config
	resource := v1alpha1.Resource{
		Attributes: c.Spec.Resource.Attributes,
	}

	if c.Spec.Resource.K8sMetadata != nil && c.Spec.Resource.K8sMetadata.IncludeUIDs {
		resource.AddK8sUIDAttributes = true
	}

	defaults := v1alpha1.Defaults{}
	if c.Spec.Resource.ServiceMetadata != nil && c.Spec.Resource.ServiceMetadata.Enabled != nil {
		defaults.UseLabelsForResourceAttributes = *c.Spec.Resource.ServiceMetadata.Enabled
	}

	result := v1alpha1.Instrumentation{
		ObjectMeta: c.ObjectMeta,
		Status: v1alpha1.InstrumentationStatus{
			UpgradeBlockedVersions: c.Status.UpgradeBlockedVersions,
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter:    exporter,
			Resource:    resource,
			Propagators: propagators,
			Sampler:     sampler,
			Defaults:    defaults,
			Env:         c.Spec.Env,
			Java: v1alpha1.Java{
				Image:               c.Spec.Java.Image,
				InitContainer:       convertInitContainerToV1alpha1(c.Spec.Java.InitContainer),
				Env:                 c.Spec.Java.Env,
				VolumeClaimTemplate: c.Spec.Java.VolumeClaimTemplate,
				Resources:           c.Spec.Java.Resources,
				Extensions:          convertExtensionsToV1alpha1(c.Spec.Java.Extensions),
			},
			NodeJS: v1alpha1.NodeJS{
				Image:               c.Spec.NodeJS.Image,
				InitContainer:       convertInitContainerToV1alpha1(c.Spec.NodeJS.InitContainer),
				Env:                 c.Spec.NodeJS.Env,
				VolumeClaimTemplate: c.Spec.NodeJS.VolumeClaimTemplate,
				Resources:           c.Spec.NodeJS.Resources,
			},
			Python: v1alpha1.Python{
				Image:               c.Spec.Python.Image,
				InitContainer:       convertInitContainerToV1alpha1(c.Spec.Python.InitContainer),
				Env:                 c.Spec.Python.Env,
				VolumeClaimTemplate: c.Spec.Python.VolumeClaimTemplate,
				Resources:           c.Spec.Python.Resources,
			},
			DotNet: v1alpha1.DotNet{
				Image:               c.Spec.DotNet.Image,
				InitContainer:       convertInitContainerToV1alpha1(c.Spec.DotNet.InitContainer),
				Env:                 c.Spec.DotNet.Env,
				VolumeClaimTemplate: c.Spec.DotNet.VolumeClaimTemplate,
				Resources:           c.Spec.DotNet.Resources,
			},
			Go: v1alpha1.Go{
				Image:               c.Spec.Go.Image,
				Env:                 c.Spec.Go.Env,
				VolumeClaimTemplate: c.Spec.Go.VolumeClaimTemplate,
				Resources:           c.Spec.Go.Resources,
				SecurityContext:     c.Spec.Go.SecurityContext,
			},
			ApacheHttpd: v1alpha1.ApacheHttpd{
				Image:               c.Spec.ApacheHttpd.Image,
				Env:                 c.Spec.ApacheHttpd.Env,
				VolumeClaimTemplate: c.Spec.ApacheHttpd.VolumeClaimTemplate,
				Resources:           c.Spec.ApacheHttpd.Resources,
				Version:             c.Spec.ApacheHttpd.Version,
				ConfigPath:          c.Spec.ApacheHttpd.ConfigPath,
			},
			Nginx: v1alpha1.Nginx{
				Image:               c.Spec.Nginx.Image,
				Env:                 c.Spec.Nginx.Env,
				VolumeClaimTemplate: c.Spec.Nginx.VolumeClaimTemplate,
				Resources:           c.Spec.Nginx.Resources,
				ConfigFile:          c.Spec.Nginx.ConfigFile,
			},
			ImagePullPolicy:              c.Spec.ImagePullPolicy,
			InitContainerSecurityContext: c.Spec.InitContainerSecurityContext,
		},
	}

	restoreV1alpha1Fields(&result)

	return result
}

func restoreV1alpha1Fields(inst *v1alpha1.Instrumentation) {
	if inst.Annotations == nil {
		return
	}

	annotationJSON, exists := inst.Annotations[v1alpha1FieldsAnnotation]
	if !exists {
		return
	}

	var lossyFields v1alpha1Fields
	if err := json.Unmarshal([]byte(annotationJSON), &lossyFields); err != nil {
		log.Printf("Warning: failed to unmarshal v1alpha1 fields annotation: %v", err)
		return
	}

	// Restore volume size limits and attrs (lossy fields)
	inst.Spec.Java.VolumeSizeLimit = lossyFields.JavaVolumeSizeLimit
	inst.Spec.NodeJS.VolumeSizeLimit = lossyFields.NodeJSVolumeSizeLimit
	inst.Spec.Python.VolumeSizeLimit = lossyFields.PythonVolumeSizeLimit
	inst.Spec.DotNet.VolumeSizeLimit = lossyFields.DotNetVolumeSizeLimit
	inst.Spec.Go.VolumeSizeLimit = lossyFields.GoVolumeSizeLimit
	inst.Spec.ApacheHttpd.VolumeSizeLimit = lossyFields.ApacheHttpdVolumeSizeLimit
	inst.Spec.ApacheHttpd.Attrs = lossyFields.ApacheHttpdAttrs
	inst.Spec.Nginx.VolumeSizeLimit = lossyFields.NginxVolumeSizeLimit
	inst.Spec.Nginx.Attrs = lossyFields.NginxAttrs

	// Delete the annotation after restoration
	delete(inst.Annotations, v1alpha1FieldsAnnotation)
}

// Helper functions

func convertInitContainerToV1beta1(in v1alpha1.InitContainer) v1beta1.InitContainer {
	return v1beta1.InitContainer{
		Command: in.Command,
		Args:    in.Args,
	}
}

func convertInitContainerToV1alpha1(in v1beta1.InitContainer) v1alpha1.InitContainer {
	return v1alpha1.InitContainer{
		Command: in.Command,
		Args:    in.Args,
	}
}

func hasLossyFields(fields v1alpha1Fields) bool {
	return fields.JavaVolumeSizeLimit != nil ||
		fields.NodeJSVolumeSizeLimit != nil ||
		fields.PythonVolumeSizeLimit != nil ||
		fields.DotNetVolumeSizeLimit != nil ||
		fields.GoVolumeSizeLimit != nil ||
		fields.ApacheHttpdVolumeSizeLimit != nil ||
		len(fields.ApacheHttpdAttrs) > 0 ||
		fields.NginxVolumeSizeLimit != nil ||
		len(fields.NginxAttrs) > 0
}

func convertTLSToV1beta1(tls *v1alpha1.TLS) *v1beta1.TLS {
	if tls == nil {
		return nil
	}
	return &v1beta1.TLS{
		SecretName:    tls.SecretName,
		ConfigMapName: tls.ConfigMapName,
		CA:            tls.CA,
		Cert:          tls.Cert,
		Key:           tls.Key,
	}
}

func convertTLSToV1alpha1(tls *v1beta1.TLS) *v1alpha1.TLS {
	if tls == nil {
		return nil
	}
	return &v1alpha1.TLS{
		SecretName:    tls.SecretName,
		ConfigMapName: tls.ConfigMapName,
		CA:            tls.CA,
		Cert:          tls.Cert,
		Key:           tls.Key,
	}
}

func convertPropagatorsToV1beta1(in []v1alpha1.Propagator) []v1beta1.Propagator {
	var result []v1beta1.Propagator
	for _, p := range in {
		result = append(result, v1beta1.Propagator(p))
	}
	return result
}

func convertPropagatorsToV1alpha1(in []v1beta1.Propagator) []v1alpha1.Propagator {
	var result []v1alpha1.Propagator
	for _, p := range in {
		result = append(result, v1alpha1.Propagator(p))
	}
	return result
}

func convertExtensionsToV1beta1(in []v1alpha1.Extensions) []v1beta1.Extensions {
	var result []v1beta1.Extensions
	for _, e := range in {
		result = append(result, v1beta1.Extensions{
			Image: e.Image,
			Dir:   e.Dir,
		})
	}
	return result
}

func convertExtensionsToV1alpha1(in []v1beta1.Extensions) []v1alpha1.Extensions {
	var result []v1alpha1.Extensions
	for _, e := range in {
		result = append(result, v1alpha1.Extensions{
			Image: e.Image,
			Dir:   e.Dir,
		})
	}
	return result
}
