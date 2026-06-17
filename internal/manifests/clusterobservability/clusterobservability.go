// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/clusterobservability/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

const (
	ComponentClusterObservability = "cluster-observability"

	// Collector name suffixes.
	AgentCollectorSuffix   = "agent"
	ClusterCollectorSuffix = "cluster"

	// Host paths mounted into the agent collector for system telemetry.
	// The host root is mounted at /hostfs for the hostmetrics receiver; container
	// logs are read from their real paths for the filelog receiver.
	hostPathRoot          = "/"
	hostPathHostfs        = "/hostfs"
	hostPathVarLogPods    = "/var/log/pods"
	hostPathDockerContain = "/var/lib/docker/containers"
)

// Build creates the manifest for the ClusterObservability resource.
func Build(params manifests.Params) ([]client.Object, error) {
	distro := config.NewConfigLoader().DetectDistroProvider(params.Config)

	var resourceManifests []client.Object

	// Build agent-level collector (DaemonSet)
	agentCollector, err := buildAgentCollector(params, distro)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent collector: %w", err)
	}
	if agentCollector != nil {
		resourceManifests = append(resourceManifests, agentCollector)
	}

	// Build cluster-level collector (StatefulSet)
	clusterCollector, err := buildClusterCollector(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster collector: %w", err)
	}
	if clusterCollector != nil {
		resourceManifests = append(resourceManifests, clusterCollector)
	}

	// Build Instrumentation CRs for all namespaces
	instrumentations, err := buildInstrumentations(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build instrumentation CRs: %w", err)
	}
	resourceManifests = append(resourceManifests, instrumentations...)

	// Build OpenShift Security Context Constraints if on OpenShift
	if distro == config.OpenShift {
		resourceManifests = append(resourceManifests, buildOpenShiftSCC(params)...)
	}

	return resourceManifests, nil
}

// buildAgentCollector creates an OpenTelemetryCollector CR for agent-level collection.
func buildAgentCollector(params manifests.Params, distro config.DistroProvider) (*v1beta1.OpenTelemetryCollector, error) {
	co := params.ClusterObservability

	// Load configuration using the config loader
	configLoader := config.NewConfigLoader()

	// Load the configuration
	collectorConfig, err := configLoader.LoadCollectorConfig(
		config.AgentCollectorType,
		distro,
		co.Spec,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent collector config: %w", err)
	}

	// Validate the configuration
	if err := configLoader.ValidateConfig(collectorConfig); err != nil {
		return nil, fmt.Errorf("agent collector config validation failed: %w", err)
	}

	agentCollectorName := fmt.Sprintf("%s-%s", co.Name, AgentCollectorSuffix)
	labels := manifestutils.Labels(co.ObjectMeta, agentCollectorName, params.Config.CollectorImage, ComponentClusterObservability, params.Config.LabelsFilter)
	labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	labels["app.kubernetes.io/component"] = ComponentClusterObservability

	agentCollector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentCollectorName,
			Namespace: co.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         co.APIVersion,
					Kind:               co.Kind,
					Name:               co.Name,
					UID:                co.UID,
					Controller:         new(true),
					BlockOwnerDeletion: new(true),
				},
			},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode:   v1beta1.ModeDaemonSet,
			Config: collectorConfig,
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image:              params.Config.ClusterObservabilityCollectorImage,
				Env:                agentEnvVars(),
				SecurityContext:    agentSecurityContext(distro),
				PodSecurityContext: agentPodSecurityContext(distro),
				HostNetwork:        true,
				VolumeMounts:       agentVolumeMounts(distro),
				Volumes:            agentVolumes(distro),
			},
		},
	}

	return agentCollector, nil
}

// buildClusterCollector creates an OpenTelemetryCollector CR for cluster-level collection.
func buildClusterCollector(params manifests.Params) (*v1beta1.OpenTelemetryCollector, error) {
	co := params.ClusterObservability

	// Load configuration using the config loader
	configLoader := config.NewConfigLoader()

	// Detect Kubernetes distribution
	distroProvider := configLoader.DetectDistroProvider(params.Config)

	// Load the configuration
	collectorConfig, err := configLoader.LoadCollectorConfig(
		config.ClusterCollectorType,
		distroProvider,
		co.Spec,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster collector config: %w", err)
	}

	// Validate the configuration
	if err := configLoader.ValidateConfig(collectorConfig); err != nil {
		return nil, fmt.Errorf("cluster collector config validation failed: %w", err)
	}

	replicas := int32(1)
	clusterCollectorName := fmt.Sprintf("%s-%s", co.Name, ClusterCollectorSuffix)
	clusterLabels := manifestutils.Labels(co.ObjectMeta, clusterCollectorName, params.Config.CollectorImage, ComponentClusterObservability, params.Config.LabelsFilter)
	clusterLabels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	clusterLabels["app.kubernetes.io/component"] = ComponentClusterObservability
	clusterLabels[constants.LabelTargetAllocator] = naming.TargetAllocator(co.Name)

	clusterCollector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterCollectorName,
			Namespace: co.Namespace,
			Labels:    clusterLabels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         co.APIVersion,
					Kind:               co.Kind,
					Name:               co.Name,
					UID:                co.UID,
					Controller:         new(true),
					BlockOwnerDeletion: new(true),
				},
			},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode:   v1beta1.ModeStatefulSet,
			Config: collectorConfig,
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image:    params.Config.ClusterObservabilityCollectorImage,
				Replicas: &replicas,
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: new(false),
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
					RunAsNonRoot: new(true),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
				PodSecurityContext: &corev1.PodSecurityContext{
					RunAsNonRoot: new(true),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
			},
		},
	}

	return clusterCollector, nil
}

// buildInstrumentations creates a single Instrumentation CR in the operator namespace
// Users can reference it via instrumentation.opentelemetry.io/ns annotation.
func buildInstrumentations(params manifests.Params) ([]client.Object, error) {
	co := params.ClusterObservability

	// Build OTLP exporter endpoint for instrumentation
	endpoint, err := buildInstrumentationEndpoint(co.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to build instrumentation endpoint: %w", err)
	}

	// Create a single Instrumentation in the same namespace as the ClusterObservability resource
	instrumentationLabels := manifestutils.Labels(co.ObjectMeta, co.Name, "", ComponentClusterObservability, params.Config.LabelsFilter)
	instrumentationLabels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	instrumentationLabels["app.kubernetes.io/component"] = ComponentClusterObservability

	instrumentation := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      co.Name,
			Namespace: co.Namespace,
			Labels:    instrumentationLabels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         co.APIVersion,
					Kind:               co.Kind,
					Name:               co.Name,
					UID:                co.UID,
					Controller:         new(true),
					BlockOwnerDeletion: new(true),
				},
			},
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: endpoint,
			},
			Propagators: []v1alpha1.Propagator{
				v1alpha1.TraceContext,
				v1alpha1.Baggage,
				v1alpha1.B3,
				v1alpha1.Jaeger,
			},
			Sampler: v1alpha1.Sampler{
				Type:     v1alpha1.ParentBasedTraceIDRatio,
				Argument: "1.0",
			},
		},
	}

	// Enable instrumentation based on operator configuration
	if params.Config.EnableJavaAutoInstrumentation {
		instrumentation.Spec.Java = v1alpha1.Java{
			Image: params.Config.AutoInstrumentationJavaImage,
		}
	}
	if params.Config.EnableNodeJSAutoInstrumentation {
		instrumentation.Spec.NodeJS = v1alpha1.NodeJS{
			Image: params.Config.AutoInstrumentationNodeJSImage,
		}
	}
	if params.Config.EnablePythonAutoInstrumentation {
		instrumentation.Spec.Python = v1alpha1.Python{
			Image: params.Config.AutoInstrumentationPythonImage,
		}
	}
	if params.Config.EnableDotNetAutoInstrumentation {
		instrumentation.Spec.DotNet = v1alpha1.DotNet{
			Image: params.Config.AutoInstrumentationDotNetImage,
		}
	}
	if params.Config.EnableGoAutoInstrumentation {
		instrumentation.Spec.Go = v1alpha1.Go{
			Image: params.Config.AutoInstrumentationGoImage,
		}
	}

	return []client.Object{instrumentation}, nil
}

// buildInstrumentationEndpoint builds the OTLP endpoint for instrumentation.
func buildInstrumentationEndpoint(v1alpha1.ClusterObservabilitySpec) (string, error) {
	return "http://$(OTEL_NODE_IP):4318", nil
}

// agentEnvVars returns the downward-API env vars referenced by the agent base config.
func agentEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "K8S_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			},
		},
	}
}

// agentSecurityContext returns the container security context for the agent.
// On OpenShift the agent must run as root with the spc_t SELinux type (supplied
// by the generated SCC) so the filelog receiver can read the root-owned,
// container_log_t-labeled files under /var/log/pods. No extra capabilities are
// required. Other distros keep the agent unprivileged.
func agentSecurityContext(distro config.DistroProvider) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{
		AllowPrivilegeEscalation: new(false),
		RunAsNonRoot:             new(true),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	if distro == config.OpenShift {
		sc.RunAsNonRoot = new(false)
		sc.RunAsUser = ptr.To[int64](0)
	}
	return sc
}

func agentPodSecurityContext(distro config.DistroProvider) *corev1.PodSecurityContext {
	psc := &corev1.PodSecurityContext{
		RunAsNonRoot:   new(true),
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	if distro == config.OpenShift {
		psc.RunAsNonRoot = new(false)
		psc.RunAsUser = ptr.To[int64](0)
	}
	return psc
}

func agentVolumeMounts(distro config.DistroProvider) []corev1.VolumeMount {
	hostToContainer := corev1.MountPropagationHostToContainer
	mounts := []corev1.VolumeMount{
		{Name: "hostfs", MountPath: hostPathHostfs, ReadOnly: true, MountPropagation: &hostToContainer},
		{Name: "var-log-pods", MountPath: hostPathVarLogPods, ReadOnly: true},
		{Name: "var-lib-docker-containers", MountPath: hostPathDockerContain, ReadOnly: true},
	}
	if distro == config.OpenShift {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "kubelet-serving-ca",
			MountPath: "/etc/kubelet-serving-ca/ca-bundle.crt",
			ReadOnly:  true,
		})
	}
	return mounts
}

func agentVolumes(distro config.DistroProvider) []corev1.Volume {
	hostPath := func(path string) corev1.VolumeSource {
		return corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path}}
	}
	volumes := []corev1.Volume{
		{Name: "hostfs", VolumeSource: hostPath(hostPathRoot)},
		{Name: "var-log-pods", VolumeSource: hostPath(hostPathVarLogPods)},
		{Name: "var-lib-docker-containers", VolumeSource: hostPath(hostPathDockerContain)},
	}
	if distro == config.OpenShift {
		fileType := corev1.HostPathFile
		volumes = append(volumes, corev1.Volume{
			Name: "kubelet-serving-ca",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/kubernetes/kubelet-ca.crt",
					Type: &fileType,
				},
			},
		})
	}
	return volumes
}
