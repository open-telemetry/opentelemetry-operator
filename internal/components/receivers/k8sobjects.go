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
)

type k8sobjectsConfig struct {
	Objects []k8sObject `yaml:"objects"`
}

type k8sObject struct {
	Name  string `yaml:"name"`
	Mode  string `yaml:"mode"`
	Group string `yaml:"group,omitempty"`
}

func generatek8sobjectsClusterRoleRules(_ logr.Logger, config k8sobjectsConfig) ([]rbacv1.PolicyRule, error) {
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8sobjectsreceiver#rbac
	prs := []rbacv1.PolicyRule{}
	for _, obj := range config.Objects {
		permissions := []string{"list"}
		if obj.Mode == "pull" && (obj.Name != "events" && obj.Name != "events.k8s.io") {
			permissions = append(permissions, "get")
		} else if obj.Mode == "watch" {
			permissions = append(permissions, "watch")
		}
		prs = append(prs, rbacv1.PolicyRule{
			APIGroups: []string{obj.Group},
			Resources: []string{obj.Name},
			Verbs:     permissions,
		})
	}
	return prs, nil
}
