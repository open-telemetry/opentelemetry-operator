// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/authorization/v1"
)

func TestWarningsGroupedByResource(t *testing.T) {
	tests := []struct {
		desc     string
		reviews  []*v1.SubjectAccessReview
		expected []string
	}{
		{
			desc:     "No access reviews",
			reviews:  nil,
			expected: nil,
		},
		{
			desc: "One access review with resource attributes",
			reviews: []*v1.SubjectAccessReview{
				{
					Spec: v1.SubjectAccessReviewSpec{
						User: "system:serviceaccount:test-ns:test-targetallocator",
						ResourceAttributes: &v1.ResourceAttributes{
							Verb:     "get",
							Group:    "",
							Resource: "namespaces",
						},
					},
				},
			},
			expected: []string{"missing the following rules for system:serviceaccount:test-ns:test-targetallocator - namespaces: [get]"},
		},
		{
			desc: "One access review with non resource attributes",
			reviews: []*v1.SubjectAccessReview{
				{
					Spec: v1.SubjectAccessReviewSpec{
						User: "system:serviceaccount:test-ns:test-targetallocator",
						ResourceAttributes: &v1.ResourceAttributes{
							Verb:     "watch",
							Group:    "apps",
							Resource: "replicasets",
						},
					},
				},
			},
			expected: []string{"missing the following rules for system:serviceaccount:test-ns:test-targetallocator - apps/replicasets: [watch]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			w := WarningsGroupedByResource(tt.reviews)
			assert.Equal(t, tt.expected, w)
		})
	}

}
