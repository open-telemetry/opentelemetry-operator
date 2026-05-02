// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"slices"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"

	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

// targetAllocatorCRPolicyRules are the policy rules required for the CR functionality.
var targetAllocatorCRPolicyRules = []*rbacv1.PolicyRule{
	{
		APIGroups: []string{""},
		Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods", "namespaces"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}, {
		APIGroups: []string{"discovery.k8s.io"},
		Resources: []string{"endpointslices"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		APIGroups: []string{"networking.k8s.io"},
		Resources: []string{"ingresses"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		NonResourceURLs: []string{"/metrics"},
		Verbs:           []string{"get"},
	}, {
		NonResourceURLs: []string{"/api", "/api/*", "/apis", "/apis/*"},
		Verbs:           []string{"get"},
	},
}

var prometheusCRDNames = []string{"servicemonitors", "podmonitors", "probes", "scrapeconfigs"}

// checkPrometheusCRDExists reports whether resourceName is served under monitoring.coreos.com.
func checkPrometheusCRDExists(dcl discovery.DiscoveryInterface, resourceName string) (bool, error) {
	if dcl == nil {
		return false, errors.New("discovery client is nil")
	}
	groups, err := dcl.ServerGroups()
	if err != nil {
		return false, err
	}
	for _, g := range groups.Groups {
		if g.Name != "monitoring.coreos.com" {
			continue
		}
		for _, v := range g.Versions {
			resources, err := dcl.ServerResourcesForGroupVersion(v.GroupVersion)
			if err != nil {
				return false, err
			}
			if slices.ContainsFunc(resources.APIResources, func(r metav1.APIResource) bool {
				return r.Name == resourceName
			}) {
				return true, nil
			}
		}
	}
	return false, nil
}

func CheckTargetAllocatorPrometheusCRPolicyRules(
	ctx context.Context,
	reviewer *rbac.Reviewer,
	dcl discovery.DiscoveryInterface,
	namespace string,
	serviceAccountName string,
) (warnings []string, err error) {
	rules := targetAllocatorCRPolicyRules

	var existingCRDs []string
	for _, name := range prometheusCRDNames {
		exists, checkErr := checkPrometheusCRDExists(dcl, name)
		if checkErr != nil {
			return []string{}, fmt.Errorf("unable to check CRD existence: %w", checkErr)
		}
		if exists {
			existingCRDs = append(existingCRDs, name)
		}
	}
	if len(existingCRDs) > 0 {
		rules = append(rules, &rbacv1.PolicyRule{
			APIGroups: []string{"monitoring.coreos.com"},
			Resources: existingCRDs,
			Verbs:     []string{"get", "list", "watch"},
		})
	}

	subjectAccessReviews, err := reviewer.CheckPolicyRules(
		ctx,
		serviceAccountName,
		namespace,
		rules...,
	)
	if err != nil {
		return []string{}, fmt.Errorf("unable to check rbac rules %w", err)
	}
	if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return []string{}, nil
}
