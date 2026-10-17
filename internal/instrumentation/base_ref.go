// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

const (
	envOTELExporterOTLPTracesEndpoint           = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	envOTELExporterOTLPMetricsEndpoint          = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"
	envOTELExporterOTLPLogsEndpoint             = "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"
	envOTELExporterOTLPTracesCertificate        = "OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE"
	envOTELExporterOTLPMetricsCertificate       = "OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE"
	envOTELExporterOTLPLogsCertificate          = "OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE"
	envOTELExporterOTLPTracesClientCertificate  = "OTEL_EXPORTER_OTLP_TRACES_CLIENT_CERTIFICATE"
	envOTELExporterOTLPMetricsClientCertificate = "OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE"
	envOTELExporterOTLPLogsClientCertificate    = "OTEL_EXPORTER_OTLP_LOGS_CLIENT_CERTIFICATE"
	envOTELExporterOTLPTracesClientKey          = "OTEL_EXPORTER_OTLP_TRACES_CLIENT_KEY"
	envOTELExporterOTLPMetricsClientKey         = "OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY"
	envOTELExporterOTLPLogsClientKey            = "OTEL_EXPORTER_OTLP_LOGS_CLIENT_KEY"
)

func (pm *instPodMutator) resolveInstrumentation(ctx context.Context, child *v1alpha1.Instrumentation) (*v1alpha1.Instrumentation, error) {
	if child.Spec.BaseRef == nil {
		return child, nil
	}

	ref := child.Spec.BaseRef
	namespace := ref.Namespace
	if namespace == "" {
		namespace = child.Namespace
	}
	parentName := types.NamespacedName{Namespace: namespace, Name: ref.Name}
	childName := types.NamespacedName{Namespace: child.Namespace, Name: child.Name}
	if parentName == childName {
		return nil, fmt.Errorf("instrumentation %s cannot reference itself", childName)
	}

	parent := &v1alpha1.Instrumentation{}
	if err := pm.Client.Get(ctx, parentName, parent); err != nil {
		return nil, fmt.Errorf("failed to get base instrumentation %s for %s: %w", parentName, childName, err)
	}
	if parent.Spec.BaseRef != nil {
		return nil, fmt.Errorf("base instrumentation %s must not set spec.baseRef", parentName)
	}

	result, err := overlayInstrumentation(parent, child)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve instrumentation %s with base %s: %w", childName, parentName, err)
	}
	return result, nil
}

// overlayInstrumentation creates an admission-only copy. Neither stored CR is modified.
func overlayInstrumentation(parent, child *v1alpha1.Instrumentation) (*v1alpha1.Instrumentation, error) {
	result := child.DeepCopy()
	base := parent.DeepCopy().Spec
	override := result.Spec
	base.BaseRef = nil

	// The base TLS credentials may not be valid for a different endpoint.
	if override.Endpoint != "" {
		base.Exporter = override.Exporter
	} else if override.TLS != nil {
		if !strings.HasPrefix(base.Endpoint, "https://") {
			return nil, errors.New("base spec.exporter.endpoint must use https:// when overriding only spec.exporter.tls")
		}
		base.TLS = override.TLS
	}
	if override.Argument != "" && override.Type == "" {
		return nil, errors.New("spec.sampler.type is required when overriding spec.sampler.argument")
	}
	if override.Type != "" {
		base.Sampler = override.Sampler
	}
	if len(override.Propagators) > 0 {
		base.Propagators = override.Propagators
	}
	if override.Defaults.UseLabelsForResourceAttributes {
		base.Defaults.UseLabelsForResourceAttributes = true
	}
	if override.Resource.AddK8sUIDAttributes {
		base.Resource.AddK8sUIDAttributes = true
	}
	if len(override.Resource.Attributes) > 0 {
		if base.Resource.Attributes == nil {
			base.Resource.Attributes = make(map[string]string, len(override.Resource.Attributes))
		}
		maps.Copy(base.Resource.Attributes, override.Resource.Attributes)
	}
	if override.ImagePullPolicy != "" {
		base.ImagePullPolicy = override.ImagePullPolicy
	}
	if override.InitContainerSecurityContext != nil {
		base.InitContainerSecurityContext = override.InitContainerSecurityContext
	}

	// The child's structured SDK settings take precedence over equivalent base environment variables.
	blocked := childTypedEnvNames(override)
	var err error
	base.Env, err = overlayEnv(base.Env, override.Env, blocked, override.Resource.Attributes)
	if err != nil {
		return nil, err
	}
	for _, env := range override.Env {
		blocked[env.Name] = struct{}{}
	}

	java, err := overlayAgent(agentSettingsFromJava(base.Java), agentSettingsFromJava(override.Java), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.java.env: %w", err)
	}
	base.Java.Image, base.Java.VolumeClaimTemplate, base.Java.VolumeSizeLimit, base.Java.Env, base.Java.Resources = java.Image, java.VolumeClaimTemplate, java.VolumeSizeLimit, java.Env, java.Resources
	if len(override.Java.Extensions) > 0 {
		base.Java.Extensions = override.Java.Extensions
	}

	nodejs, err := overlayAgent(agentSettingsFromNodeJS(base.NodeJS), agentSettingsFromNodeJS(override.NodeJS), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.nodejs.env: %w", err)
	}
	base.NodeJS.Image, base.NodeJS.VolumeClaimTemplate, base.NodeJS.VolumeSizeLimit, base.NodeJS.Env, base.NodeJS.Resources = nodejs.Image, nodejs.VolumeClaimTemplate, nodejs.VolumeSizeLimit, nodejs.Env, nodejs.Resources

	python, err := overlayAgent(agentSettingsFromPython(base.Python), agentSettingsFromPython(override.Python), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.python.env: %w", err)
	}
	base.Python.Image, base.Python.VolumeClaimTemplate, base.Python.VolumeSizeLimit, base.Python.Env, base.Python.Resources = python.Image, python.VolumeClaimTemplate, python.VolumeSizeLimit, python.Env, python.Resources

	dotnet, err := overlayAgent(agentSettingsFromDotNet(base.DotNet), agentSettingsFromDotNet(override.DotNet), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.dotnet.env: %w", err)
	}
	base.DotNet.Image, base.DotNet.VolumeClaimTemplate, base.DotNet.VolumeSizeLimit, base.DotNet.Env, base.DotNet.Resources = dotnet.Image, dotnet.VolumeClaimTemplate, dotnet.VolumeSizeLimit, dotnet.Env, dotnet.Resources

	goAgent, err := overlayAgent(agentSettingsFromGo(base.Go), agentSettingsFromGo(override.Go), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.go.env: %w", err)
	}
	base.Go.Image, base.Go.VolumeClaimTemplate, base.Go.VolumeSizeLimit, base.Go.Env, base.Go.Resources = goAgent.Image, goAgent.VolumeClaimTemplate, goAgent.VolumeSizeLimit, goAgent.Env, goAgent.Resources
	if override.Go.SecurityContext != nil {
		base.Go.SecurityContext = override.Go.SecurityContext
	}

	apache, err := overlayAgent(agentSettingsFromApache(base.ApacheHttpd), agentSettingsFromApache(override.ApacheHttpd), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.apacheHttpd.env: %w", err)
	}
	base.ApacheHttpd.Image, base.ApacheHttpd.VolumeClaimTemplate, base.ApacheHttpd.VolumeSizeLimit, base.ApacheHttpd.Env, base.ApacheHttpd.Resources = apache.Image, apache.VolumeClaimTemplate, apache.VolumeSizeLimit, apache.Env, apache.Resources
	base.ApacheHttpd.Attrs, err = overlayEnv(base.ApacheHttpd.Attrs, override.ApacheHttpd.Attrs, nil, nil)
	if err != nil {
		return nil, err
	}
	if override.ApacheHttpd.Version != "" {
		base.ApacheHttpd.Version = override.ApacheHttpd.Version
	}
	if override.ApacheHttpd.ConfigPath != "" {
		base.ApacheHttpd.ConfigPath = override.ApacheHttpd.ConfigPath
	}

	nginx, err := overlayAgent(agentSettingsFromNginx(base.Nginx), agentSettingsFromNginx(override.Nginx), blocked, override.Resource.Attributes)
	if err != nil {
		return nil, fmt.Errorf("spec.nginx.env: %w", err)
	}
	base.Nginx.Image, base.Nginx.VolumeClaimTemplate, base.Nginx.VolumeSizeLimit, base.Nginx.Env, base.Nginx.Resources = nginx.Image, nginx.VolumeClaimTemplate, nginx.VolumeSizeLimit, nginx.Env, nginx.Resources
	base.Nginx.Attrs, err = overlayEnv(base.Nginx.Attrs, override.Nginx.Attrs, nil, nil)
	if err != nil {
		return nil, err
	}
	if override.Nginx.ConfigFile != "" {
		base.Nginx.ConfigFile = override.Nginx.ConfigFile
	}

	result.Spec = base
	return result, nil
}

type agentSettings struct {
	Image               string
	VolumeClaimTemplate corev1.PersistentVolumeClaimTemplate
	VolumeSizeLimit     *resource.Quantity
	Env                 []corev1.EnvVar
	Resources           corev1.ResourceRequirements
}

func agentSettingsFromJava(spec v1alpha1.Java) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func agentSettingsFromNodeJS(spec v1alpha1.NodeJS) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func agentSettingsFromPython(spec v1alpha1.Python) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func agentSettingsFromDotNet(spec v1alpha1.DotNet) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func agentSettingsFromGo(spec v1alpha1.Go) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func agentSettingsFromApache(spec v1alpha1.ApacheHttpd) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func agentSettingsFromNginx(spec v1alpha1.Nginx) agentSettings {
	return agentSettings{spec.Image, spec.VolumeClaimTemplate, spec.VolumeSizeLimit, spec.Env, spec.Resources}
}

func overlayAgent(base, child agentSettings, blocked map[string]struct{}, childAttrs map[string]string) (agentSettings, error) {
	if child.Image != "" {
		base.Image = child.Image
	}
	if !reflect.ValueOf(child.VolumeClaimTemplate).IsZero() {
		base.VolumeClaimTemplate = child.VolumeClaimTemplate
		base.VolumeSizeLimit = nil
	} else if child.VolumeSizeLimit != nil {
		base.VolumeClaimTemplate = corev1.PersistentVolumeClaimTemplate{}
		base.VolumeSizeLimit = child.VolumeSizeLimit
	}
	var err error
	base.Env, err = overlayEnv(base.Env, child.Env, blocked, childAttrs)
	if err != nil {
		return agentSettings{}, err
	}
	base.Resources = overlayResources(base.Resources, child.Resources)
	return base, nil
}

func overlayResources(base, child corev1.ResourceRequirements) corev1.ResourceRequirements {
	if len(child.Limits) > 0 {
		if base.Limits == nil {
			base.Limits = corev1.ResourceList{}
		}
		maps.Copy(base.Limits, child.Limits)
	}
	if len(child.Requests) > 0 {
		if base.Requests == nil {
			base.Requests = corev1.ResourceList{}
		}
		maps.Copy(base.Requests, child.Requests)
	}
	if len(child.Claims) > 0 {
		base.Claims = child.Claims
	}
	return base
}

func childTypedEnvNames(child v1alpha1.InstrumentationSpec) map[string]struct{} {
	blocked := map[string]struct{}{}
	if child.Endpoint != "" || child.TLS != nil {
		for _, name := range []string{
			constants.EnvOTELExporterOTLPEndpoint,
			envOTELExporterOTLPTracesEndpoint,
			envOTELExporterOTLPMetricsEndpoint,
			envOTELExporterOTLPLogsEndpoint,
			constants.EnvOTELExporterCertificate,
			constants.EnvOTELExporterClientCertificate,
			constants.EnvOTELExporterClientKey,
			envOTELExporterOTLPTracesCertificate,
			envOTELExporterOTLPMetricsCertificate,
			envOTELExporterOTLPLogsCertificate,
			envOTELExporterOTLPTracesClientCertificate,
			envOTELExporterOTLPMetricsClientCertificate,
			envOTELExporterOTLPLogsClientCertificate,
			envOTELExporterOTLPTracesClientKey,
			envOTELExporterOTLPMetricsClientKey,
			envOTELExporterOTLPLogsClientKey,
		} {
			blocked[name] = struct{}{}
		}
	}
	if child.Type != "" {
		blocked[constants.EnvOTELTracesSampler] = struct{}{}
		blocked[constants.EnvOTELTracesSamplerArg] = struct{}{}
	}
	if len(child.Propagators) > 0 {
		blocked[constants.EnvOTELPropagators] = struct{}{}
	}
	return blocked
}

func overlayEnv(base, child []corev1.EnvVar, blocked map[string]struct{}, childAttrs map[string]string) ([]corev1.EnvVar, error) {
	result := append([]corev1.EnvVar(nil), child...)
	seen := make(map[string]struct{}, len(child))
	for _, env := range child {
		seen[env.Name] = struct{}{}
	}
	for _, env := range base {
		if _, ok := seen[env.Name]; ok {
			continue
		}
		if _, ok := blocked[env.Name]; ok {
			continue
		}
		if env.Name == constants.EnvOTELResourceAttrs && len(childAttrs) > 0 {
			var err error
			env, err = removeOverriddenResourceAttributes(env, childAttrs)
			if err != nil {
				return nil, err
			}
			if env.Value == "" {
				continue
			}
		}
		result = append(result, env)
		seen[env.Name] = struct{}{}
	}
	return result, nil
}

func removeOverriddenResourceAttributes(env corev1.EnvVar, childAttrs map[string]string) (corev1.EnvVar, error) {
	if env.ValueFrom != nil {
		return corev1.EnvVar{}, fmt.Errorf("cannot merge %s from valueFrom with child resource attributes", env.Name)
	}
	if env.Value == "" {
		return env, nil
	}
	var kept []string
	for entry := range strings.SplitSeq(env.Value, ",") {
		key, _, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok || key == "" || strings.Contains(key, "\\") {
			return corev1.EnvVar{}, fmt.Errorf("cannot merge malformed %s with child resource attributes", env.Name)
		}
		if _, overridden := childAttrs[key]; !overridden {
			kept = append(kept, entry)
		}
	}
	env.Value = strings.Join(kept, ",")
	return env, nil
}
