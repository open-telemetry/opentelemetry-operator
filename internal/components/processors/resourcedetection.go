// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processors

import (
	"fmt"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ResourceDetectionConfig is a minimal struct needed for parsing a valid resourcedetection processor configuration
// This only contains the fields necessary for parsing, other fields can be added in the future.
type ResourceDetectionConfig struct {
	Detectors []string `mapstructure:"detectors"`
}

func generateResourceDetectionClusterRoleRules(_ logr.Logger, config ResourceDetectionConfig) ([]rbacv1.PolicyRule, error) {
	var prs []rbacv1.PolicyRule
	for _, d := range config.Detectors {
		detectorName := fmt.Sprint(d)
		switch detectorName {
		case "k8snode":
			policy := rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list"},
			}
			prs = append(prs, policy)
		case "openshift":
			policy := rbacv1.PolicyRule{
				APIGroups: []string{"config.openshift.io"},
				Resources: []string{"infrastructures", "infrastructures/status"},
				Verbs:     []string{"get", "watch", "list"},
			}
			prs = append(prs, policy)
		}
	}
	return prs, nil
}
