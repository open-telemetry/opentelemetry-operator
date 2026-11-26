// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

func TestCheckCertManagerPermissions(t *testing.T) {
	t.Setenv(autodetectutils.NAMESPACE_ENV_VAR, "operator-system")
	t.Setenv(autodetectutils.SA_ENV_VAR, "operator")
	tests := []struct {
		name         string
		status       authorizationv1.SubjectAccessReviewStatus
		err          error
		wantErr      bool
		wantWarnings bool
	}{
		{name: "allowed", status: authorizationv1.SubjectAccessReviewStatus{Allowed: true}},
		{name: "denied", status: authorizationv1.SubjectAccessReviewStatus{Denied: true}, wantWarnings: true},
		{name: "no opinion", wantWarnings: true},
		{name: "evaluation error", status: authorizationv1.SubjectAccessReviewStatus{EvaluationError: "authorizer unavailable"}, wantErr: true},
		{name: "allowed despite evaluation error", status: authorizationv1.SubjectAccessReviewStatus{Allowed: true, EvaluationError: "another authorizer unavailable"}},
		{name: "request failed", err: errors.New("API unavailable"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewClientset()
			clientset.PrependReactor("create", "subjectaccessreviews", func(action kubeTesting.Action) (bool, runtime.Object, error) {
				review := action.(kubeTesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview).DeepCopy()
				review.Status = tt.status
				return true, review, tt.err
			})
			warnings, err := CheckCertManagerPermissions(context.Background(), rbac.NewReviewer(clientset))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantWarnings, len(warnings) > 0)
		})
	}
}
