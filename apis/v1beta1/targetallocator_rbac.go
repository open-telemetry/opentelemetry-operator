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

package v1beta1

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

var (

	// targetAllocatorCRPolicyRules are the policy rules required for the CR functionality.
	targetAllocatorCRPolicyRules = []*rbacv1.PolicyRule{
		{
			APIGroups: []string{"monitoring.coreos.com"},
			Resources: []string{"servicemonitors", "podmonitors"},
			Verbs:     []string{"*"},
		}, {
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
		},
	}
)

func CheckTargetAllocatorPrometheusCRPolicyRules(
	ctx context.Context,
	reviewer *rbac.Reviewer,
	namespace string,
	serviceAccountName string) (warnings []string, err error) {
	subjectAccessReviews, err := reviewer.CheckPolicyRules(
		ctx,
		namespace,
		serviceAccountName,
		targetAllocatorCRPolicyRules...,
	)
	if err != nil {
		return []string{}, fmt.Errorf("unable to check rbac rules %w", err)
	}
	if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return []string{}, nil
}
