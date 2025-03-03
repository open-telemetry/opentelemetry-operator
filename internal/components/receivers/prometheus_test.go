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
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratePrometheusReceiverRoles(t *testing.T) {
	tests := []struct {
		name          string
		config        prometheusReceiverConfig
		componentName string
		want          []*rbacv1.Role
	}{
		{
			name: "nil config",
			config: prometheusReceiverConfig{
				Config: nil,
			},
			componentName: "component",
			want:          nil,
		},
		{
			name: "nil scrape configs",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: nil,
				},
			},
			componentName: "component",
			want:          nil,
		},
		{
			name: "single pod role with multiple namespaces",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: &[]scrapeConfig{
						{
							JobName: "job",
							KubernetesSDConfigs: []kubernetesSDConfig{
								{
									Role: "pod",
									Namespaces: namespaces{
										Names: []string{"ns1", "ns2"},
									},
								},
							},
						},
					},
				},
			},
			componentName: "component",
			want: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job-component-role",
						Namespace: "ns1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "watch", "list"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job-component-role",
						Namespace: "ns2",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "watch", "list"},
						},
					},
				},
			},
		},
		{
			name: "multiple roles and namespaces",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: &[]scrapeConfig{
						{
							JobName: "job",
							KubernetesSDConfigs: []kubernetesSDConfig{
								{
									Role: "pod",
									Namespaces: namespaces{
										Names: []string{"ns1"},
									},
								},
								{
									Role: "service",
									Namespaces: namespaces{
										Names: []string{"ns2"},
									},
								},
							},
						},
					},
				},
			},
			componentName: "component",
			want: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job-component-role",
						Namespace: "ns1",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "watch", "list"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-job-component-role",
						Namespace: "ns2",
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"services"},
							Verbs:     []string{"get", "watch", "list"},
						},
					},
				},
			},
		},
		{
			name: "unsupported role type",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: &[]scrapeConfig{
						{
							JobName: "job",
							KubernetesSDConfigs: []kubernetesSDConfig{
								{
									Role: "unsupported",
									Namespaces: namespaces{
										Names: []string{"ns1"},
									},
								},
							},
						},
					},
				},
			},
			componentName: "component",
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testr.New(t)
			got, err := generatePrometheusReceiverRoles(logger, tt.config, tt.componentName, "test")
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGeneratePrometheusReceiverRoleBindings(t *testing.T) {
	namespace := "test-ns"
	tests := []struct {
		name               string
		config             prometheusReceiverConfig
		componentName      string
		serviceAccountName string
		want               int // number of expected role bindings
	}{
		{
			name: "nil config",
			config: prometheusReceiverConfig{
				Config: nil,
			},
			componentName:      "test-component",
			serviceAccountName: "test-sa",
			want:               0,
		},
		{
			name: "nil scrape configs",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: nil,
				},
			},
			componentName:      "test-component",
			serviceAccountName: "test-sa",
			want:               0,
		},
		{
			name: "single namespace and job",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: &[]scrapeConfig{
						{
							JobName: "test-job",
							KubernetesSDConfigs: []kubernetesSDConfig{
								{
									Role: "pod",
									Namespaces: namespaces{
										Names: []string{"test-ns"},
									},
								},
							},
						},
					},
				},
			},
			componentName:      "test-component",
			serviceAccountName: "test-sa",
			want:               1,
		},
		{
			name: "multiple namespaces and jobs",
			config: prometheusReceiverConfig{
				Config: &prometheusConfig{
					ScrapeConfigs: &[]scrapeConfig{
						{
							JobName: "test-job-1",
							KubernetesSDConfigs: []kubernetesSDConfig{
								{
									Role: "pod",
									Namespaces: namespaces{
										Names: []string{"test-ns-1", "test-ns-2"},
									},
								},
							},
						},
						{
							JobName: "test-job-2",
							KubernetesSDConfigs: []kubernetesSDConfig{
								{
									Role: "service",
									Namespaces: namespaces{
										Names: []string{"test-ns-3"},
									},
								},
							},
						},
					},
				},
			},
			componentName:      "test-component",
			serviceAccountName: "test-sa",
			want:               3,
		},
	}

	logger := logr.Discard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generatePrometheusReceiverRoleBindings(logger, tt.config, tt.componentName, tt.serviceAccountName, "test", namespace)
			if err != nil {
				t.Errorf("generatePrometheusReceiverRoleBindings() error = %v", err)
				return
			}

			if len(got) != tt.want {
				t.Errorf("generatePrometheusReceiverRoleBindings() got %d role bindings, want %d", len(got), tt.want)
			}

			// For non-empty results, verify the role binding properties
			if len(got) > 0 {
				rb := got[0]
				if rb.Subjects[0].Name != tt.serviceAccountName {
					t.Errorf("Role binding has wrong service account name, got %s, want %s",
						rb.Subjects[0].Name, tt.serviceAccountName)
				}
				if rb.Subjects[0].Kind != rbacv1.ServiceAccountKind {
					t.Errorf("Role binding has wrong subject kind, got %s, want %s",
						rb.Subjects[0].Kind, rbacv1.ServiceAccountKind)
				}
				if rb.RoleRef.Kind != "Role" {
					t.Errorf("Role binding has wrong role ref kind, got %s, want Role",
						rb.RoleRef.Kind)
				}
			}
		})
	}
}
