// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processors

import (
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

type K8sAttributeFilter struct {
	Namespace string `mapstructure:"namespace"`
}

// K8sAttributeConfig is a minimal struct needed for parsing a valid k8sattribute processor configuration
// This only contains the fields necessary for parsing, other fields can be added in the future.
type K8sAttributeConfig struct {
	Filter  K8sAttributeFilter `mapstructure:"filter"`
	Extract Extract            `mapstructure:"extract"`
}

// generateK8SAttrPolicyRules builds the RBAC policy rules needed by the
// k8sattributes processor based on its extract configuration.
func generateK8SAttrPolicyRules(config K8sAttributeConfig) []rbacv1.PolicyRule {
	prs := []rbacv1.PolicyRule{
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
		metadataField := m
		if (metadataField == "k8s.deployment.uid" || metadataField == "k8s.deployment.name" || metadataField == "service.name") && !addedReplicasetPolicy {
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
	return prs
}

// GenerateK8SAttrRbacRules returns cluster-scoped RBAC rules for the
// k8sattributes processor. When filter.namespace is set, the rules are
// namespace-scoped instead and this returns nil.
func GenerateK8SAttrRbacRules(_ logr.Logger, config K8sAttributeConfig) ([]rbacv1.PolicyRule, error) {
	if config.Filter.Namespace != "" {
		return nil, nil
	}
	return generateK8SAttrPolicyRules(config), nil
}

// GenerateK8SAttrNamespacedRbacRules returns namespace-scoped RBAC rules for
// the k8sattributes processor. When filter.namespace is set, the rules are
// scoped to that namespace (Role instead of ClusterRole).
func GenerateK8SAttrNamespacedRbacRules(_ logr.Logger, config K8sAttributeConfig) (map[string][]rbacv1.PolicyRule, error) {
	if config.Filter.Namespace == "" {
		return nil, nil
	}
	return map[string][]rbacv1.PolicyRule{
		config.Filter.Namespace: generateK8SAttrPolicyRules(config),
	}, nil
}
