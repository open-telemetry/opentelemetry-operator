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

package receivers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type metricConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type metrics struct {
	K8sContainerCPULimitUtilization      metricConfig `mapstructure:"k8s.container.cpu_limit_utilization"`
	K8sContainerCPURequestUtilization    metricConfig `mapstructure:"k8s.container.cpu_request_utilization"`
	K8sContainerMemoryLimitUtilization   metricConfig `mapstructure:"k8s.container.memory_limit_utilization"`
	K8sContainerMemoryRequestUtilization metricConfig `mapstructure:"k8s.container.memory_request_utilization"`
	K8sPodCPULimitUtilization            metricConfig `mapstructure:"k8s.pod.cpu_limit_utilization"`
	K8sPodCPURequestUtilization          metricConfig `mapstructure:"k8s.pod.cpu_request_utilization"`
	K8sPodMemoryLimitUtilization         metricConfig `mapstructure:"k8s.pod.memory_limit_utilization"`
	K8sPodMemoryRequestUtilization       metricConfig `mapstructure:"k8s.pod.memory_request_utilization"`
}

// KubeletStatsConfig is a minimal struct needed for parsing a valid kubeletstats receiver configuration
// This only contains the fields necessary for parsing, other fields can be added in the future.
type kubeletStatsConfig struct {
	ExtraMetadataLabels []string `mapstructure:"extra_metadata_labels"`
	Metrics             metrics  `mapstructure:"metrics"`
	AuthType            string   `mapstructure:"auth_type"`
}

func generateKubeletStatsEnvVars(_ logr.Logger, config kubeletStatsConfig) ([]corev1.EnvVar, error) {
	// The documentation mentions that the K8S_NODE_NAME environment variable is required when using the serviceAccount auth type.
	// Also, it mentions that it is a good idea to use it for the Read Only Endpoint. Added always to make it easier for users.
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/kubeletstatsreceiver/README.md
	return []corev1.EnvVar{
		{Name: "K8S_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
	}, nil
}

func generateKubeletStatsClusterRoleRules(_ logr.Logger, config kubeletStatsConfig) ([]rbacv1.PolicyRule, error) {
	// The Kubelet Stats Receiver needs get permissions on the nodes/stats resources always.
	prs := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes/stats"},
			Verbs:     []string{"get"},
		},
	}

	// Additionally, when using extra_metadata_labels or any of the {request|limit}_utilization metrics
	// the processor also needs get permissions for nodes/proxy resources.
	nodesProxyPr := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"nodes/proxy"},
		Verbs:     []string{"get"},
	}

	if len(config.ExtraMetadataLabels) > 0 {
		prs = append(prs, nodesProxyPr)
		return prs, nil
	}

	metrics := []bool{
		config.Metrics.K8sContainerCPULimitUtilization.Enabled,
		config.Metrics.K8sContainerCPURequestUtilization.Enabled,
		config.Metrics.K8sContainerMemoryLimitUtilization.Enabled,
		config.Metrics.K8sContainerMemoryRequestUtilization.Enabled,
		config.Metrics.K8sPodCPULimitUtilization.Enabled,
		config.Metrics.K8sPodCPURequestUtilization.Enabled,
		config.Metrics.K8sPodMemoryLimitUtilization.Enabled,
		config.Metrics.K8sPodMemoryRequestUtilization.Enabled,
	}
	for _, metric := range metrics {
		if metric {
			prs = append(prs, nodesProxyPr)
			return prs, nil
		}
	}
	return prs, nil
}
