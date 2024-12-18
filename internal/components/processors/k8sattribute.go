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

package processors

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

type FieldExtractConfig struct {
	TagName  string `mapstructure:"tag_name"`
	Key      string `mapstructure:"key"`
	KeyRegex string `mapstructure:"key_regex"`
	Regex    string `mapstructure:"regex"`
	From     string `mapstructure:"from"`
}

type Extract struct {
	Metadata    []string             `mapstructure:"metadata"`
	Labels      []FieldExtractConfig `mapstructure:"labels"`
	Annotations []FieldExtractConfig `mapstructure:"annotations"`
}

// K8sAttributeConfig is a minimal struct needed for parsing a valid k8sattribute processor configuration
// This only contains the fields necessary for parsing, other fields can be added in the future.
type K8sAttributeConfig struct {
	Extract Extract `mapstructure:"extract"`
}

func generateK8SAttrClusterRoleRules(_ logr.Logger, config K8sAttributeConfig) ([]rbacv1.PolicyRule, error) {
	// These policies need to be added always
	var prs = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "namespaces"},
			Verbs:     []string{"get", "watch", "list"},
		},
	}

	replicasetPolicy := rbacv1.PolicyRule{
		APIGroups: []string{"apps"},
		Resources: []string{"replicasets"},
		Verbs:     []string{"get", "watch", "list"},
	}

	if len(config.Extract.Metadata) == 0 {
		prs = append(prs, replicasetPolicy)
	}
	addedReplicasetPolicy := false
	for _, m := range config.Extract.Metadata {
		metadataField := fmt.Sprint(m)
		if (metadataField == "k8s.deployment.uid" || metadataField == "k8s.deployment.name") && !addedReplicasetPolicy {
			prs = append(prs, replicasetPolicy)
			addedReplicasetPolicy = true
		} else if strings.Contains(metadataField, "k8s.node") {
			prs = append(prs,
				rbacv1.PolicyRule{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "watch", "list"},
				},
			)
		}
	}
	return prs, nil
}
