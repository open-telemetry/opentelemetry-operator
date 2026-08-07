// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const v1alpha1FieldsAnnotation = "instrumentation.opentelemetry.io/v1alpha1-fields"

// v1alpha1Fields holds lossy fields that don't map directly to v1beta1.
type v1alpha1Fields struct {
	JavaEnv                    []corev1.EnvVar    `json:"javaEnv,omitempty"`
	JavaVolumeSizeLimit        *resource.Quantity `json:"javaVolumeSizeLimit,omitempty"`
	NodeJSEnv                  []corev1.EnvVar    `json:"nodejsEnv,omitempty"`
	NodeJSVolumeSizeLimit      *resource.Quantity `json:"nodejsVolumeSizeLimit,omitempty"`
	PythonEnv                  []corev1.EnvVar    `json:"pythonEnv,omitempty"`
	PythonVolumeSizeLimit      *resource.Quantity `json:"pythonVolumeSizeLimit,omitempty"`
	DotNetEnv                  []corev1.EnvVar    `json:"dotnetEnv,omitempty"`
	DotNetVolumeSizeLimit      *resource.Quantity `json:"dotnetVolumeSizeLimit,omitempty"`
	GoEnv                      []corev1.EnvVar    `json:"goEnv,omitempty"`
	GoVolumeSizeLimit          *resource.Quantity `json:"goVolumeSizeLimit,omitempty"`
	ApacheHttpdEnv             []corev1.EnvVar    `json:"apacheHttpdEnv,omitempty"`
	ApacheHttpdVolumeSizeLimit *resource.Quantity `json:"apacheHttpdVolumeSizeLimit,omitempty"`
	ApacheHttpdAttrs           []corev1.EnvVar    `json:"apacheHttpdAttrs,omitempty"`
	NginxEnv                   []corev1.EnvVar    `json:"nginxEnv,omitempty"`
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

	// Merge all env vars into envConfig.env
	mergedEnv := mergeEnvVars(c.Spec.Env, c.Spec.Java.Env, c.Spec.NodeJS.Env, c.Spec.Python.Env,
		c.Spec.DotNet.Env, c.Spec.Go.Env, c.Spec.ApacheHttpd.Env, c.Spec.Nginx.Env)

	// Build envConfig
	var envConfig *v1beta1.EnvConfig
	if c.Spec.Endpoint != "" || len(c.Spec.Propagators) > 0 ||
		c.Spec.Type != "" || len(mergedEnv) > 0 {
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
			Env: mergedEnv,
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
		JavaEnv:                    c.Spec.Java.Env,
		JavaVolumeSizeLimit:        c.Spec.Java.VolumeSizeLimit,
		NodeJSEnv:                  c.Spec.NodeJS.Env,
		NodeJSVolumeSizeLimit:      c.Spec.NodeJS.VolumeSizeLimit,
		PythonEnv:                  c.Spec.Python.Env,
		PythonVolumeSizeLimit:      c.Spec.Python.VolumeSizeLimit,
		DotNetEnv:                  c.Spec.DotNet.Env,
		DotNetVolumeSizeLimit:      c.Spec.DotNet.VolumeSizeLimit,
		GoEnv:                      c.Spec.Go.Env,
		GoVolumeSizeLimit:          c.Spec.Go.VolumeSizeLimit,
		ApacheHttpdEnv:             c.Spec.ApacheHttpd.Env,
		ApacheHttpdVolumeSizeLimit: c.Spec.ApacheHttpd.VolumeSizeLimit,
		ApacheHttpdAttrs:           c.Spec.ApacheHttpd.Attrs,
		NginxEnv:                   c.Spec.Nginx.Env,
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
			Java: v1beta1.Java{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Java.Image,
					VolumeClaimTemplate: c.Spec.Java.VolumeClaimTemplate,
					Resources:           c.Spec.Java.Resources,
				},
				Extensions: convertExtensionsToV1beta1(c.Spec.Java.Extensions),
			},
			NodeJS: v1beta1.NodeJS{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.NodeJS.Image,
					VolumeClaimTemplate: c.Spec.NodeJS.VolumeClaimTemplate,
					Resources:           c.Spec.NodeJS.Resources,
				},
			},
			Python: v1beta1.Python{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Python.Image,
					VolumeClaimTemplate: c.Spec.Python.VolumeClaimTemplate,
					Resources:           c.Spec.Python.Resources,
				},
			},
			DotNet: v1beta1.DotNet{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.DotNet.Image,
					VolumeClaimTemplate: c.Spec.DotNet.VolumeClaimTemplate,
					Resources:           c.Spec.DotNet.Resources,
				},
			},
			Go: v1beta1.Go{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Go.Image,
					VolumeClaimTemplate: c.Spec.Go.VolumeClaimTemplate,
					Resources:           c.Spec.Go.Resources,
				},
				SecurityContext: c.Spec.Go.SecurityContext,
			},
			ApacheHttpd: v1beta1.ApacheHttpd{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.ApacheHttpd.Image,
					VolumeClaimTemplate: c.Spec.ApacheHttpd.VolumeClaimTemplate,
					Resources:           c.Spec.ApacheHttpd.Resources,
				},
				Version:    c.Spec.ApacheHttpd.Version,
				ConfigPath: c.Spec.ApacheHttpd.ConfigPath,
			},
			Nginx: v1beta1.Nginx{
				CommonLanguageSpec: v1beta1.CommonLanguageSpec{
					Image:               c.Spec.Nginx.Image,
					VolumeClaimTemplate: c.Spec.Nginx.VolumeClaimTemplate,
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
	var commonEnv []corev1.EnvVar

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
		commonEnv = c.Spec.EnvConfig.Env
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
			Env:         commonEnv,
			Java: v1alpha1.Java{
				Image:               c.Spec.Java.Image,
				VolumeClaimTemplate: c.Spec.Java.VolumeClaimTemplate,
				Resources:           c.Spec.Java.Resources,
				Extensions:          convertExtensionsToV1alpha1(c.Spec.Java.Extensions),
			},
			NodeJS: v1alpha1.NodeJS{
				Image:               c.Spec.NodeJS.Image,
				VolumeClaimTemplate: c.Spec.NodeJS.VolumeClaimTemplate,
				Resources:           c.Spec.NodeJS.Resources,
			},
			Python: v1alpha1.Python{
				Image:               c.Spec.Python.Image,
				VolumeClaimTemplate: c.Spec.Python.VolumeClaimTemplate,
				Resources:           c.Spec.Python.Resources,
			},
			DotNet: v1alpha1.DotNet{
				Image:               c.Spec.DotNet.Image,
				VolumeClaimTemplate: c.Spec.DotNet.VolumeClaimTemplate,
				Resources:           c.Spec.DotNet.Resources,
			},
			Go: v1alpha1.Go{
				Image:               c.Spec.Go.Image,
				VolumeClaimTemplate: c.Spec.Go.VolumeClaimTemplate,
				Resources:           c.Spec.Go.Resources,
				SecurityContext:     c.Spec.Go.SecurityContext,
			},
			ApacheHttpd: v1alpha1.ApacheHttpd{
				Image:               c.Spec.ApacheHttpd.Image,
				VolumeClaimTemplate: c.Spec.ApacheHttpd.VolumeClaimTemplate,
				Resources:           c.Spec.ApacheHttpd.Resources,
				Version:             c.Spec.ApacheHttpd.Version,
				ConfigPath:          c.Spec.ApacheHttpd.ConfigPath,
			},
			Nginx: v1alpha1.Nginx{
				Image:               c.Spec.Nginx.Image,
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

	// Restore per-language env and volume size limits
	inst.Spec.Java.Env = lossyFields.JavaEnv
	inst.Spec.Java.VolumeSizeLimit = lossyFields.JavaVolumeSizeLimit
	inst.Spec.NodeJS.Env = lossyFields.NodeJSEnv
	inst.Spec.NodeJS.VolumeSizeLimit = lossyFields.NodeJSVolumeSizeLimit
	inst.Spec.Python.Env = lossyFields.PythonEnv
	inst.Spec.Python.VolumeSizeLimit = lossyFields.PythonVolumeSizeLimit
	inst.Spec.DotNet.Env = lossyFields.DotNetEnv
	inst.Spec.DotNet.VolumeSizeLimit = lossyFields.DotNetVolumeSizeLimit
	inst.Spec.Go.Env = lossyFields.GoEnv
	inst.Spec.Go.VolumeSizeLimit = lossyFields.GoVolumeSizeLimit
	inst.Spec.ApacheHttpd.Env = lossyFields.ApacheHttpdEnv
	inst.Spec.ApacheHttpd.VolumeSizeLimit = lossyFields.ApacheHttpdVolumeSizeLimit
	inst.Spec.ApacheHttpd.Attrs = lossyFields.ApacheHttpdAttrs
	inst.Spec.Nginx.Env = lossyFields.NginxEnv
	inst.Spec.Nginx.VolumeSizeLimit = lossyFields.NginxVolumeSizeLimit
	inst.Spec.Nginx.Attrs = lossyFields.NginxAttrs

	// Remove per-language envs from Spec.Env to recover the original common-only list.
	// During v1alpha1→v1beta1, per-language envs were merged into envConfig.env;
	// now that they've been restored to their per-language fields, strip them from Spec.Env.
	inst.Spec.Env = subtractEnvVars(inst.Spec.Env,
		lossyFields.JavaEnv, lossyFields.NodeJSEnv, lossyFields.PythonEnv,
		lossyFields.DotNetEnv, lossyFields.GoEnv, lossyFields.ApacheHttpdEnv,
		lossyFields.NginxEnv)

	// Delete the annotation after restoration
	delete(inst.Annotations, v1alpha1FieldsAnnotation)
}

// Helper functions

func hasLossyFields(fields v1alpha1Fields) bool {
	return len(fields.JavaEnv) > 0 || fields.JavaVolumeSizeLimit != nil ||
		len(fields.NodeJSEnv) > 0 || fields.NodeJSVolumeSizeLimit != nil ||
		len(fields.PythonEnv) > 0 || fields.PythonVolumeSizeLimit != nil ||
		len(fields.DotNetEnv) > 0 || fields.DotNetVolumeSizeLimit != nil ||
		len(fields.GoEnv) > 0 || fields.GoVolumeSizeLimit != nil ||
		len(fields.ApacheHttpdEnv) > 0 || fields.ApacheHttpdVolumeSizeLimit != nil ||
		len(fields.ApacheHttpdAttrs) > 0 ||
		len(fields.NginxEnv) > 0 || fields.NginxVolumeSizeLimit != nil ||
		len(fields.NginxAttrs) > 0
}

func subtractEnvVars(all []corev1.EnvVar, toRemove ...[]corev1.EnvVar) []corev1.EnvVar {
	removeCounts := make(map[string]int)
	for _, list := range toRemove {
		for _, e := range list {
			removeCounts[e.Name]++
		}
	}

	keep := make([]bool, len(all))
	for i, v := range slices.Backward(all) {
		if count := removeCounts[v.Name]; count > 0 {
			removeCounts[v.Name]--
		} else {
			keep[i] = true
		}
	}

	var result []corev1.EnvVar
	for i, e := range all {
		if keep[i] {
			result = append(result, e)
		}
	}
	return result
}

func mergeEnvVars(envLists ...[]corev1.EnvVar) []corev1.EnvVar {
	var result []corev1.EnvVar
	for _, envList := range envLists {
		result = append(result, envList...)
	}
	return result
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
