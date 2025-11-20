// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
)

const (
	OpenShiftSCCAPIVersion = "security.openshift.io/v1"
	OpenShiftSCCKind       = "SecurityContextConstraints"
)

// buildOpenShiftSCC creates SecurityContextConstraints for agent collectors
// TODO: Use structured API https://pkg.go.dev/github.com/openshift/api/security/v1#SecurityContextConstraints
func buildOpenShiftSCC(params manifests.Params) []client.Object {
	co := params.ClusterObservability

	// Only create SCC for agent collector (needs host access for metrics and logs)
	agentCollectorName := fmt.Sprintf("%s-%s", co.Name, AgentCollectorSuffix)
	sccName := fmt.Sprintf("%s-hostaccess", agentCollectorName)

	labels := manifestutils.Labels(co.ObjectMeta, sccName, "", ComponentClusterObservability, params.Config.LabelsFilter)
	labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	labels["app.kubernetes.io/component"] = ComponentClusterObservability

	// SCC configuration for host access
	scc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": OpenShiftSCCAPIVersion,
			"kind":       OpenShiftSCCKind,
			"metadata": map[string]interface{}{
				"name":   sccName,
				"labels": labels,
				"annotations": map[string]interface{}{
					"kubernetes.io/description": "Allows OpenTelemetry agent collectors to access host resources for metrics and log collection",
				},
			},
			"priority":                 10,
			"allowHostDirVolumePlugin": true,
			"allowHostNetwork":         true,
			"allowHostPID":             true,
			"allowHostPorts":           true,
			"allowHostIPC":             false,
			"allowPrivilegedContainer": false,
			"readOnlyRootFilesystem":   true,
			// SELinux context for podman/crio socket and /proc access
			"seLinuxContext": map[string]interface{}{
				"type": "MustRunAs",
				"seLinuxOptions": map[string]interface{}{
					"user":  "system_u",
					"role":  "system_r",
					"type":  "spc_t",
					"level": "s0",
				},
			},
			"runAsUser": map[string]interface{}{
				"type": "RunAsAny",
			},
			"fsGroup": map[string]interface{}{
				"type": "RunAsAny",
			},
			"supplementalGroups": map[string]interface{}{
				"type": "RunAsAny",
			},
			"allowedCapabilities":      []string{},
			"defaultAddCapabilities":   []string{},
			"requiredDropCapabilities": []string{"ALL"},
			"seccompProfiles":          []string{"runtime/default"},
			"allowedUnsafeSysctls":     []string{},
			"forbiddenSysctls":         []string{},
			"volumes": []string{
				"configMap",
				"downwardAPI",
				"emptyDir",
				"hostPath",
				"projected",
				"secret",
			},
			"users": []string{
				// Reference the ServiceAccount that OpenTelemetryCollector controller creates
				fmt.Sprintf("system:serviceaccount:%s:%s-collector", co.Namespace, agentCollectorName),
			},
		},
	}

	scc.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "security.openshift.io",
		Version: "v1",
		Kind:    "SecurityContextConstraints",
	})

	return []client.Object{scc}
}
