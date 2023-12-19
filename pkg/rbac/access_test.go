package rbac

import (
	"context"
	"fmt"
	"testing"

	v1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"
)

const (
	createVerb  = "create"
	sarResource = "subjectaccessreviews"
)

type fakeClientGenerator func() kubernetes.Interface

func reactorFactory(status v1.SubjectAccessReviewStatus, mockErr error) fakeClientGenerator {
	return func() kubernetes.Interface {
		c := fake.NewSimpleClientset()
		c.PrependReactor(createVerb, sarResource, func(action kubeTesting.Action) (handled bool, ret runtime.Object, err error) {
			// check our expectation here
			if !action.Matches(createVerb, sarResource) {
				return false, nil, fmt.Errorf("must be a create for a SAR")
			}
			sar, ok := action.(kubeTesting.CreateAction).GetObject().DeepCopyObject().(*v1.SubjectAccessReview)
			if !ok || sar == nil {
				return false, nil, fmt.Errorf("bad object")
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
			name: "can't access",
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
			clientGenerator: reactorFactory(v1.SubjectAccessReviewStatus{}, fmt.Errorf("failed to create SAR")),
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
			got, err := r.CanAccess(context.Background(), tt.args.serviceAccount, tt.args.serviceAccountNamespace, tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CanAccess() got = %v, want %v", got, tt.want)
			}
		})
	}
}
