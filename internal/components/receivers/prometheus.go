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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

type namespaces struct {
	Names []string `mapstructure:"names"`
}

type kubernetesSDConfig struct {
	Namespaces namespaces `mapstructure:"namespaces"`
	Role       string     `mapstructure:"role"`
}

type scrapeConfig struct {
	KubernetesSDConfigs []kubernetesSDConfig `mapstructure:"kubernetes_sd_configs"`
	JobName             string               `mapstructure:"job_name"`
}

type prometheusConfig struct {
	ScrapeConfigs *[]scrapeConfig `mapstructure:"scrape_configs"`
}

type prometheusReceiverConfig struct {
	Config *prometheusConfig `mapstructure:"config"`
}

func generatePrometheusReceiverRoles(logger logr.Logger, config prometheusReceiverConfig, componentName string, otelCollectorName string) ([]*rbacv1.Role, error) {
	if config.Config == nil {
		return nil, nil
	}

	if config.Config.ScrapeConfigs == nil {
		return nil, nil
	}

	var roles []*rbacv1.Role

	for _, scrapeConfig := range *config.Config.ScrapeConfigs {
		for _, kubernetesSDConfig := range scrapeConfig.KubernetesSDConfigs {
			var rule rbacv1.PolicyRule
			switch kubernetesSDConfig.Role {
			case "pod":
				rule = rbacv1.PolicyRule{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "watch", "list"},
				}
			case "node":
				rule = rbacv1.PolicyRule{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "watch", "list"},
				}
			case "service":
				rule = rbacv1.PolicyRule{
					APIGroups: []string{""},
					Resources: []string{"services"},
					Verbs:     []string{"get", "watch", "list"},
				}
			case "endpoints":
				rule = rbacv1.PolicyRule{
					APIGroups: []string{""},
					Resources: []string{"endpoints", "services"},
					Verbs:     []string{"get", "watch", "list"},
				}
			case "ingress":
				rule = rbacv1.PolicyRule{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{"ingresses"},
					Verbs:     []string{"get", "watch", "list"},
				}
			default:
				logger.Info("unsupported role used for prometheus receiver", "role", kubernetesSDConfig.Role)
				continue
			}

			for _, namespace := range kubernetesSDConfig.Namespaces.Names {
				// We need to create a role for each namespace and role
				roles = append(roles, &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      getRoleName(scrapeConfig.JobName, componentName, otelCollectorName),
						Namespace: namespace,
					},
					Rules: []rbacv1.PolicyRule{rule},
				})
			}
		}
	}
	return roles, nil
}

func generatePrometheusReceiverRoleBindings(logger logr.Logger, config prometheusReceiverConfig, componentName string, serviceAccountName string, otelCollectorName string, otelCollectorNamespace string) ([]*rbacv1.RoleBinding, error) {
	if config.Config == nil {
		return nil, nil
	}

	if config.Config.ScrapeConfigs == nil {
		return nil, nil
	}

	var roleBindings []*rbacv1.RoleBinding

	for _, scrapeConfig := range *config.Config.ScrapeConfigs {
		for _, kubernetesSDConfig := range scrapeConfig.KubernetesSDConfigs {
			for _, namespace := range kubernetesSDConfig.Namespaces.Names {

				rb := rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      getRoleBindingName(scrapeConfig.JobName, componentName, otelCollectorName),
						Namespace: namespace,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      rbacv1.ServiceAccountKind,
							Name:      serviceAccountName,
							Namespace: otelCollectorNamespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     getRoleName(scrapeConfig.JobName, componentName, otelCollectorName),
					},
				}

				roleBindings = append(roleBindings, &rb)
			}
		}
	}

	return roleBindings, nil
}

func getRoleName(jobName string, componentName string, otelCollectorName string) string {
	return naming.Role(otelCollectorName, jobName+"-"+componentName)
}

func getRoleBindingName(jobName string, componentName string, otelCollectorName string) string {
	return naming.RoleBinding(otelCollectorName, jobName+"-"+componentName)
}
