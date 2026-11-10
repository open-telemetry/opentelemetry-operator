// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

func TestCheckCertManagerPermissions(t *testing.T) {
	tests := []struct {
		name      string
		status    authorizationv1.SubjectAccessReviewStatus
		reviewErr error
		wantErr   string
		warnings  []string
	}{
		{name: "allowed", status: authorizationv1.SubjectAccessReviewStatus{Allowed: true}},
		{
			name:   "denied",
			status: authorizationv1.SubjectAccessReviewStatus{Denied: true},
			warnings: []string{
				"missing the following rules for system:serviceaccount:operator-system:operator - cert-manager.io/issuers: [create,get,list,watch,update,patch,delete]",
				"missing the following rules for system:serviceaccount:operator-system:operator - cert-manager.io/certificaterequests: [create,get,list,watch,update,patch,delete]",
				"missing the following rules for system:serviceaccount:operator-system:operator - cert-manager.io/certificates: [create,get,list,watch,update,patch,delete]",
			},
		},
		{name: "evaluation error", status: authorizationv1.SubjectAccessReviewStatus{EvaluationError: "authorizer unavailable"}, wantErr: "unable to evaluate cert-manager permissions: authorizer unavailable"},
		{name: "allowed despite evaluation error", status: authorizationv1.SubjectAccessReviewStatus{Allowed: true, EvaluationError: "other authorizer unavailable"}},
		{name: "review request failed", reviewErr: errors.New("API unavailable"), wantErr: "unable to check rbac rules: API unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(autodetectutils.NAMESPACE_ENV_VAR, "operator-system")
			t.Setenv(autodetectutils.SA_ENV_VAR, "operator")
			client := fake.NewClientset()
			var requests []string
			client.PrependReactor("create", "subjectaccessreviews", func(action kubeTesting.Action) (bool, runtime.Object, error) {
				review := action.(kubeTesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview).DeepCopy()
				require.Equal(t, "system:serviceaccount:operator-system:operator", review.Spec.User)
				require.Equal(t, "cert-manager.io", review.Spec.ResourceAttributes.Group)
				require.Empty(t, review.Spec.ResourceAttributes.Namespace)
				requests = append(requests, fmt.Sprintf("%s/%s", review.Spec.ResourceAttributes.Verb, review.Spec.ResourceAttributes.Resource))
				review.Status = test.status
				return true, review, test.reviewErr
			})

			warnings, err := CheckCertManagerPermissions(context.Background(), rbac.NewReviewer(client))
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Empty(t, warnings)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, test.warnings, warnings)
			}
			var expected []string
			for _, verb := range []string{"create", "get", "list", "watch", "update", "patch", "delete"} {
				for _, resource := range []string{"issuers", "certificaterequests", "certificates"} {
					expected = append(expected, fmt.Sprintf("%s/%s", verb, resource))
				}
			}
			require.ElementsMatch(t, expected, requests)
		})
	}
}

func TestCheckCertManagerPermissionsMissingServiceAccount(t *testing.T) {
	t.Setenv(autodetectutils.NAMESPACE_ENV_VAR, "operator-system")
	t.Setenv(autodetectutils.SA_ENV_VAR, "")
	client := fake.NewClientset()
	warnings, err := CheckCertManagerPermissions(context.Background(), rbac.NewReviewer(client))
	require.ErrorContains(t, err, "SERVICE_ACCOUNT_NAME env variable not found")
	require.Empty(t, warnings)
	require.Empty(t, client.Actions())
}
