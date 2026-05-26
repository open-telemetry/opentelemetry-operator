// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"
)

// captureReactor is a reactor that records which SubjectAccessReview specs were submitted.
type captureReactor struct {
	status   v1.SubjectAccessReviewStatus
	mockErr  error
	captured []*v1.SubjectAccessReview
}

func (cr *captureReactor) react(action kubeTesting.Action) (handled bool, ret runtime.Object, err error) {
	if !action.Matches(createVerb, sarResource) {
		return false, nil, errors.New("must be a create for a SAR")
	}
	sar, ok := action.(kubeTesting.CreateAction).GetObject().DeepCopyObject().(*v1.SubjectAccessReview)
	if !ok || sar == nil {
		return false, nil, errors.New("bad object")
	}
	cr.captured = append(cr.captured, sar.DeepCopy())
	sar.Status = cr.status
	return true, sar, cr.mockErr
}

const (
	createVerb  = "create"
	sarResource = "subjectaccessreviews"
)

type fakeClientGenerator func() kubernetes.Interface

func reactorFactory(status v1.SubjectAccessReviewStatus, mockErr error) fakeClientGenerator {
	return func() kubernetes.Interface {
		c := fake.NewClientset()
		c.PrependReactor(createVerb, sarResource, func(action kubeTesting.Action) (handled bool, ret runtime.Object, err error) {
			// check our expectation here
			if !action.Matches(createVerb, sarResource) {
				return false, nil, errors.New("must be a create for a SAR")
			}
			sar, ok := action.(kubeTesting.CreateAction).GetObject().DeepCopyObject().(*v1.SubjectAccessReview)
			if !ok || sar == nil {
				return false, nil, errors.New("bad object")
			}
			sar.Status = status
			return true, sar, mockErr
		})
		return c
	}
}

func TestReviewer_CanAccess(t *testing.T) {
	type args struct {
		serviceAccount          string
		serviceAccountNamespace string
		res                     *v1.ResourceAttributes
	}
	tests := []struct {
		name            string
		clientGenerator fakeClientGenerator
		args            args
		want            bool
		wantErr         bool
	}{
		{
			name: "cannot access",
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{
				Denied: true,
			}, nil),
			args: args{
				serviceAccount:          "test",
				serviceAccountNamespace: "default",
				res: &v1.ResourceAttributes{
					Namespace: "",
					Verb:      "list",
					Resource:  "namespaces",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "can access",
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{
				Allowed: true,
			}, nil),
			args: args{
				serviceAccount:          "test",
				serviceAccountNamespace: "default",
				res: &v1.ResourceAttributes{
					Namespace: "",
					Verb:      "list",
					Resource:  "namespaces",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "handles error",
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{}, errors.New("failed to create SAR")),
			args: args{
				serviceAccount:          "test",
				serviceAccountNamespace: "default",
				res: &v1.ResourceAttributes{
					Namespace: "",
					Verb:      "list",
					Resource:  "namespaces",
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewer(tt.clientGenerator())
			got, err := r.CanAccess(context.Background(), tt.args.serviceAccount, tt.args.serviceAccountNamespace, tt.args.res, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Status.Denied && got.Status.Denied != !tt.want {
				assert.Equal(t, tt.want, got.Status.Denied)
			} else if got.Status.Allowed != tt.want {
				assert.Equal(t, tt.want, got.Status.Allowed)
			}
		})
	}
}

func TestReviewer_CheckPolicyRules(t *testing.T) {
	type args struct {
		serviceAccount          string
		serviceAccountNamespace string
		policyRules             []*rbacv1.PolicyRule
	}
	tests := []struct {
		name             string
		clientGenerator  fakeClientGenerator
		args             args
		want             bool
		wantErr          bool
		numFailedReviews int
	}{
		{
			name: "cannot access",
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{
				Denied: true,
			}, nil),
			args: args{
				serviceAccount:          "test",
				serviceAccountNamespace: "default",
				policyRules: []*rbacv1.PolicyRule{
					{
						Verbs:     []string{"get", "list", "watch"},
						APIGroups: []string{""},
						Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods"},
					},
					{
						Verbs:           []string{"get"},
						NonResourceURLs: []string{"/metrics"},
					},
				},
			},
			want:             false,
			wantErr:          false,
			numFailedReviews: 16,
		},
		{
			name: "can access",
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{
				Allowed: true,
			}, nil),
			args: args{
				serviceAccount:          "test",
				serviceAccountNamespace: "default",
				policyRules: []*rbacv1.PolicyRule{
					{
						Verbs:     []string{"get", "list", "watch"},
						APIGroups: []string{""},
						Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods"},
					},
					nil, // check that we handle nil policy rules
					{
						Verbs:           []string{"get"},
						NonResourceURLs: []string{"/metrics"},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "handles error",
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{}, errors.New("failed to create SAR")),
			args: args{
				serviceAccount:          "test",
				serviceAccountNamespace: "default",
				policyRules: []*rbacv1.PolicyRule{
					{
						Verbs:     []string{"get", "list", "watch"},
						APIGroups: []string{""},
						Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods"},
					},
					{
						Verbs:           []string{"get"},
						NonResourceURLs: []string{"/metrics"},
					},
				},
			},
			want:             false,
			wantErr:          true,
			numFailedReviews: 16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReviewer(tt.clientGenerator())
			got, err := r.CheckPolicyRules(context.Background(), tt.args.serviceAccount, tt.args.serviceAccountNamespace, tt.args.policyRules...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPolicyRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			ok, deniedReviews := AllSubjectAccessReviewsAllowed(got)
			assert.Equal(t, tt.want, ok)
			if !ok {
				assert.Equal(t, tt.numFailedReviews, len(deniedReviews))
			}
		})
	}
}

func TestReviewer_CanAccessAsUser(t *testing.T) {
	tests := []struct {
		name      string
		status    v1.SubjectAccessReviewStatus
		mockErr   error
		username  string
		groups    []string
		res       *v1.ResourceAttributes
		wantAllow bool
		wantErr   bool
	}{
		{
			name:     "user can access",
			status:   v1.SubjectAccessReviewStatus{Allowed: true},
			username: "alice",
			groups:   []string{"devs"},
			res: &v1.ResourceAttributes{
				Verb:     "list",
				Resource: "nodes",
			},
			wantAllow: true,
		},
		{
			name:     "user cannot access",
			status:   v1.SubjectAccessReviewStatus{Denied: true},
			username: "bob",
			groups:   []string{"viewers"},
			res: &v1.ResourceAttributes{
				Verb:     "list",
				Resource: "nodes",
			},
			wantAllow: false,
		},
		{
			name:    "handles error",
			mockErr: errors.New("api unavailable"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &captureReactor{status: tt.status, mockErr: tt.mockErr}
			c := fake.NewClientset()
			c.PrependReactor(createVerb, sarResource, cr.react)
			r := NewReviewer(c)

			got, err := r.CanAccessAsUser(context.Background(), tt.username, tt.groups, tt.res, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanAccessAsUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			assert.Equal(t, tt.wantAllow, got.Status.Allowed)

			// Verify the SAR was submitted with the correct user fields (not the SA form).
			require.Len(t, cr.captured, 1)
			assert.Equal(t, tt.username, cr.captured[0].Spec.User)
			assert.Equal(t, tt.groups, cr.captured[0].Spec.Groups)
			// The SA format "system:serviceaccount:..." must NOT appear.
			assert.NotContains(t, cr.captured[0].Spec.User, "system:serviceaccount")
		})
	}
}

func TestReviewer_CheckSARsForUser(t *testing.T) {
	// Build two "denied SA SARs" with known resource attributes — these simulate the output
	// of CheckPolicyRules when the SA is missing permissions.
	deniedSARs := []*v1.SubjectAccessReview{
		{Spec: v1.SubjectAccessReviewSpec{ResourceAttributes: &v1.ResourceAttributes{Verb: "get", Resource: "events"}}},
		{Spec: v1.SubjectAccessReviewSpec{ResourceAttributes: &v1.ResourceAttributes{Verb: "list", Resource: "events"}}},
	}

	tests := []struct {
		name      string
		status    v1.SubjectAccessReviewStatus
		mockErr   error
		username  string
		groups    []string
		wantAllow bool
		wantErr   bool
	}{
		{
			name:      "user holds the delta permissions — allowed",
			status:    v1.SubjectAccessReviewStatus{Allowed: true},
			username:  "alice",
			groups:    []string{"platform"},
			wantAllow: true,
		},
		{
			name:      "user lacks the delta permissions — denied",
			status:    v1.SubjectAccessReviewStatus{Denied: true},
			username:  "bob",
			groups:    []string{"readonly"},
			wantAllow: false,
		},
		{
			name:    "api error propagates",
			mockErr: errors.New("api unavailable"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &captureReactor{status: tt.status, mockErr: tt.mockErr}
			c := fake.NewClientset()
			c.PrependReactor(createVerb, sarResource, cr.react)
			r := NewReviewer(c)

			got, err := r.CheckSARsForUser(context.Background(), tt.username, tt.groups, deniedSARs)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSARsForUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			ok, _ := AllSubjectAccessReviewsAllowed(got)
			assert.Equal(t, tt.wantAllow, ok)
			assert.Len(t, got, len(deniedSARs), "should produce one result SAR per input SAR")

			// Verify the re-issued SARs carry the user identity and the same resource attributes.
			require.Len(t, cr.captured, len(deniedSARs))
			for i, sar := range cr.captured {
				assert.Equal(t, tt.username, sar.Spec.User)
				assert.Equal(t, tt.groups, sar.Spec.Groups)
				assert.Equal(t, deniedSARs[i].Spec.ResourceAttributes, sar.Spec.ResourceAttributes)
			}
		})
	}
}

func TestReviewer_CheckPolicyRulesForUser(t *testing.T) {
	policyRules := []*rbacv1.PolicyRule{
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{""},
			Resources: []string{"nodes", "services"},
		},
		{
			Verbs:           []string{"get"},
			NonResourceURLs: []string{"/metrics"},
		},
	}
	// 2 resources × 3 verbs = 6 resource SARs + 1 non-resource SAR = 7 total.
	const totalSARs = 7

	tests := []struct {
		name             string
		status           v1.SubjectAccessReviewStatus
		mockErr          error
		username         string
		groups           []string
		wantAllow        bool
		wantErr          bool
		numDeniedReviews int
	}{
		{
			name:      "all rules allowed",
			status:    v1.SubjectAccessReviewStatus{Allowed: true},
			username:  "alice",
			groups:    []string{"platform"},
			wantAllow: true,
		},
		{
			name:             "all rules denied",
			status:           v1.SubjectAccessReviewStatus{Denied: true},
			username:         "bob",
			groups:           []string{"readonly"},
			wantAllow:        false,
			numDeniedReviews: totalSARs,
		},
		{
			name:             "api error propagates",
			mockErr:          errors.New("api unavailable"),
			username:         "alice",
			wantErr:          true,
			numDeniedReviews: totalSARs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &captureReactor{status: tt.status, mockErr: tt.mockErr}
			c := fake.NewClientset()
			c.PrependReactor(createVerb, sarResource, cr.react)
			r := NewReviewer(c)

			got, err := r.CheckPolicyRulesForUser(context.Background(), tt.username, tt.groups, policyRules...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPolicyRulesForUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			ok, deniedReviews := AllSubjectAccessReviewsAllowed(got)
			assert.Equal(t, tt.wantAllow, ok)
			if !ok {
				assert.Equal(t, tt.numDeniedReviews, len(deniedReviews))
			}

			// All captured SARs should carry the user identity, not the SA form.
			for _, sar := range cr.captured {
				assert.Equal(t, tt.username, sar.Spec.User)
				assert.Equal(t, tt.groups, sar.Spec.Groups)
			}
		})
	}
}
