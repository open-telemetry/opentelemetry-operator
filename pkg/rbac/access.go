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

	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	serviceAccountFmtStr = "system:serviceaccount:%s:%s"
)

type Reviewer struct {
	client kubernetes.Interface
}

func NewReviewer(c kubernetes.Interface) *Reviewer {
	return &Reviewer{
		client: c,
	}
}

// CanAccess checks if the given serviceAccount is able to access the requested resource attributes. The operator
// uses this functionality to ensure that users have the right RBAC configured for collector related service accounts.
func (r *Reviewer) CanAccess(ctx context.Context, serviceAccount, serviceAccountNamespace string, res *v1.ResourceAttributes) (bool, error) {
	sar := &v1.SubjectAccessReview{
		Spec: v1.SubjectAccessReviewSpec{
			ResourceAttributes: res,
			User:               fmt.Sprintf(serviceAccountFmtStr, serviceAccountNamespace, serviceAccount),
		},
	}
	if resp, err := r.client.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{}); err != nil {
		return false, err
	} else if resp.Status.Denied {
		return false, nil
	} else {
		return resp.Status.Allowed, nil
	}

}
