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

package rbac

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

// CheckRBACPermissions checks if the operator has the needed permissions to create RBAC resources automatically.
// If the RBAC is there, no errors nor warnings are returned.
func CheckRBACPermissions(ctx context.Context, reviewer *rbac.Reviewer) (admission.Warnings, error) {
	namespace, err := autodetectutils.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "not possible to check RBAC rules", err)
	}

	serviceAccount, err := autodetectutils.GetOperatorServiceAccount()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "not possible to check RBAC rules", err)
	}

	rules := []*rbacv1.PolicyRule{
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"clusterrolebindings", "clusterroles"},
			Verbs:     []string{"create", "delete", "get", "list", "patch", "update"},
		},
	}

	if subjectAccessReviews, err := reviewer.CheckPolicyRules(ctx, serviceAccount, namespace, rules...); err != nil {
		return nil, fmt.Errorf("%s: %w", "unable to check rbac rules", err)
	} else if allowed, deniedReviews := rbac.AllSubjectAccessReviewsAllowed(subjectAccessReviews); !allowed {
		return rbac.WarningsGroupedByResource(deniedReviews), nil
	}
	return nil, nil
}
