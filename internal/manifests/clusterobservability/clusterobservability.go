// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
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
	hostPathDev           = "/dev"
	hostPathEtc           = "/etc"
	hostPathProc          = "/proc"
	hostPathRunUdevData   = "/run/udev/data"
	hostPathSys           = "/sys"
	hostPathVarRunUtmp    = "/var/run/utmp"
	hostPathVarLogPods    = "/var/log/pods"
	hostPathDockerContain = "/var/lib/docker/containers"
	// /etc/os-release is specified by the os-release spec as the primary path
	// and exists on all Linux distributions. On some distros (e.g. OpenShift/RHCOS),
	// /usr/lib/os-release is the vendor file and /etc/os-release is a symlink to it,
	// but on others (e.g. Talos Linux) only /etc/os-release exists.
	hostPathOSRelease = "/etc/os-release"
)

// getCollectorImage returns a sensible default collector image when build-time version is not set.
func getCollectorImage(configuredImage string) string {
	// If the configured image has a 0.0.0 tag (fallback during development builds)
	// replace it with latest
	if configuredImage == "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:0.0.0" {
		return "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:latest"
	}
	return configuredImage
}

// Build creates the manifest for the ClusterObservability resource.
func Build(params manifests.Params) ([]client.Object, error) {
	var resourceManifests []client.Object

	// Build agent-level collector (DaemonSet)
	agentCollector, err := buildAgentCollector(params)
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
	if isOpenShiftEnvironment(params) {
		sccResources := buildOpenShiftSCC(params)
		resourceManifests = append(resourceManifests, sccResources...)
	}

	return resourceManifests, nil
}

// buildAgentCollector creates an OpenTelemetryCollector CR for agent-level collection.
func buildAgentCollector(params manifests.Params) (*v1beta1.OpenTelemetryCollector, error) {
	co := params.ClusterObservability

	// Load configuration using the config loader
	configLoader := config.NewConfigLoader()

	// Detect Kubernetes distribution
	distroProvider := configLoader.DetectDistroProvider(params.Config)

	// Load the configuration
	collectorConfig, err := configLoader.LoadCollectorConfig(
		config.AgentCollectorType,
		distroProvider,
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
					Controller:         &[]bool{true}[0],
					BlockOwnerDeletion: &[]bool{true}[0],
				},
			},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode:   v1beta1.ModeDaemonSet,
			Config: collectorConfig,
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image: getCollectorImage(params.Config.CollectorImage),
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: &[]bool{false}[0],
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
					RunAsNonRoot: &[]bool{true}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
				PodSecurityContext: &corev1.PodSecurityContext{
					RunAsNonRoot: &[]bool{true}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
				// Enable host networking for DaemonSet to allow direct port access
				HostNetwork: true,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "host-dev",
						MountPath: "/hostfs" + hostPathDev,
						ReadOnly:  true,
					},
					{
						Name:      "host-etc",
						MountPath: "/hostfs" + hostPathEtc,
						ReadOnly:  true,
					},
					{
						Name:      "host-proc",
						MountPath: "/hostfs" + hostPathProc,
						ReadOnly:  true,
					},
					{
						Name:      "host-run-udev-data",
						MountPath: "/hostfs" + hostPathRunUdevData,
						ReadOnly:  true,
					},
					{
						Name:      "host-sys",
						MountPath: "/hostfs" + hostPathSys,
						ReadOnly:  true,
					},
					{
						Name:      "host-var-run-utmp",
						MountPath: "/hostfs" + hostPathVarRunUtmp,
						ReadOnly:  true,
					},
					{
						Name:      "host-etc-osrelease",
						MountPath: "/hostfs" + hostPathOSRelease,
						ReadOnly:  true,
					},
					{
						Name:      "var-log-pods",
						MountPath: hostPathVarLogPods,
						ReadOnly:  true,
					},
					{
						Name:      "var-lib-docker-containers",
						MountPath: hostPathDockerContain,
						ReadOnly:  true,
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "host-dev",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathDev,
							},
						},
					},
					{
						Name: "host-etc",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathEtc,
							},
						},
					},
					{
						Name: "host-proc",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathProc,
							},
						},
					},
					{
						Name: "host-run-udev-data",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathRunUdevData,
							},
						},
					},
					{
						Name: "host-sys",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathSys,
							},
						},
					},
					{
						Name: "host-var-run-utmp",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathVarRunUtmp,
							},
						},
					},
					{
						Name: "host-etc-osrelease",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathOSRelease,
							},
						},
					},
					{
						Name: "var-log-pods",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathVarLogPods,
							},
						},
					},
					{
						Name: "var-lib-docker-containers",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: hostPathDockerContain,
							},
						},
					},
				},
			},
		},
	}

	if isOpenShiftEnvironment(params) {
		agentCollector.Spec.VolumeMounts = append(agentCollector.Spec.VolumeMounts, corev1.VolumeMount{
			Name:      "kubelet-serving-ca",
			MountPath: "/etc/kubelet-serving-ca/ca-bundle.crt",
			ReadOnly:  true,
		})
		agentCollector.Spec.Volumes = append(agentCollector.Spec.Volumes, corev1.Volume{
			Name: "kubelet-serving-ca",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/kubernetes/kubelet-ca.crt",
					Type: &[]corev1.HostPathType{corev1.HostPathFile}[0],
				},
			},
		})
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
					Controller:         &[]bool{true}[0],
					BlockOwnerDeletion: &[]bool{true}[0],
				},
			},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode:   v1beta1.ModeStatefulSet,
			Config: collectorConfig,
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image:    getCollectorImage(params.Config.CollectorImage),
				Replicas: &replicas,
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: &[]bool{false}[0],
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
					RunAsNonRoot: &[]bool{true}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
				PodSecurityContext: &corev1.PodSecurityContext{
					RunAsNonRoot: &[]bool{true}[0],
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
					Controller:         &[]bool{true}[0],
					BlockOwnerDeletion: &[]bool{true}[0],
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
	// Point to local node's agent collector
	endpoint := "http://$(OTEL_NODE_IP):4317"

	return endpoint, nil
}

// isOpenShiftEnvironment detects if we're running in an OpenShift environment using cached config.
func isOpenShiftEnvironment(params manifests.Params) bool {
	return params.Config.OpenShiftRoutesAvailability == openshift.RoutesAvailable
}
