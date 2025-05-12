// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	serviceAccountFmtStr = "system:serviceaccount:%s:%s"
)

type SAReviewer interface {
	CheckPolicyRules(ctx context.Context, serviceAccount, serviceAccountNamespace string, rules ...*rbacv1.PolicyRule) ([]*v1.SubjectAccessReview, error)
	CanAccess(ctx context.Context, serviceAccount, serviceAccountNamespace string, res *v1.ResourceAttributes, nonResourceAttributes *v1.NonResourceAttributes) (*v1.SubjectAccessReview, error)
}

var _ SAReviewer = &Reviewer{}

type Reviewer struct {
	client kubernetes.Interface
}

func NewReviewer(c kubernetes.Interface) *Reviewer {
	return &Reviewer{
		client: c,
	}
}

// AllSubjectAccessReviewsAllowed checks if all of subjectAccessReviews are explicitly allowed. If false, the method
// returns the reviews that were denied.
func AllSubjectAccessReviewsAllowed(subjectAccessReviews []*v1.SubjectAccessReview) (bool, []*v1.SubjectAccessReview) {
	allowed := true
	var deniedReviews []*v1.SubjectAccessReview
	for _, review := range subjectAccessReviews {
		if review.Status.Denied {
			allowed = false
			deniedReviews = append(deniedReviews, review)
		} else if !review.Status.Allowed {
			allowed = false
			deniedReviews = append(deniedReviews, review)
		}
	}
	return allowed, deniedReviews
}

// CheckPolicyRules is a convenience function that lets the caller check access for a set of PolicyRules.
func (r *Reviewer) CheckPolicyRules(ctx context.Context, serviceAccount, serviceAccountNamespace string, rules ...*rbacv1.PolicyRule) ([]*v1.SubjectAccessReview, error) {
	var subjectAccessReviews []*v1.SubjectAccessReview
	var errs []error
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		resourceAttributes := policyRuleToResourceAttributes(rule)
		nonResourceAttributes := policyRuleToNonResourceAttributes(rule)
		for _, res := range resourceAttributes {
			sar, err := r.CanAccess(ctx, serviceAccount, serviceAccountNamespace, res, nil)
			subjectAccessReviews = append(subjectAccessReviews, sar)
			errs = append(errs, err)
		}
		for _, nonResourceAttribute := range nonResourceAttributes {
			sar, err := r.CanAccess(ctx, serviceAccount, serviceAccountNamespace, nil, nonResourceAttribute)
			subjectAccessReviews = append(subjectAccessReviews, sar)
			errs = append(errs, err)
		}
	}
	return subjectAccessReviews, errors.Join(errs...)
}

// CanAccess checks if the given serviceAccount is able to access a single requested resource attribute.
// The operator uses this functionality to ensure that users have the right RBAC configured for collector
// related service accounts.
func (r *Reviewer) CanAccess(ctx context.Context, serviceAccount, serviceAccountNamespace string, res *v1.ResourceAttributes, nonResourceAttributes *v1.NonResourceAttributes) (*v1.SubjectAccessReview, error) {
	sar := &v1.SubjectAccessReview{
		Spec: v1.SubjectAccessReviewSpec{
			ResourceAttributes:    res,
			NonResourceAttributes: nonResourceAttributes,
			User:                  fmt.Sprintf(serviceAccountFmtStr, serviceAccountNamespace, serviceAccount),
		},
	}
	return r.client.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
}

// policyRuleToResourceAttributes converts a single policy rule in to a list of resource attribute requests.
// policyRules have lists of resources, verbs, groups, etc. whereas resource attributes do not work on lists. This
// requires us to iterate over each list and flatten.
func policyRuleToResourceAttributes(rule *rbacv1.PolicyRule) []*v1.ResourceAttributes {
	var resourceAttributes []*v1.ResourceAttributes
	for _, verb := range rule.Verbs {
		for _, group := range rule.APIGroups {
			for _, resource := range rule.Resources {
				res := &v1.ResourceAttributes{
					Verb:     verb,
					Group:    group,
					Resource: resource,
				}
				resourceAttributes = append(resourceAttributes, res)
			}
		}
	}
	return resourceAttributes
}

// policyRuleToResourceAttributes converts a single policy rule in to a list of resource attribute requests.
// policyRules have lists of resources, verbs, groups, etc. whereas resource attributes do not work on lists. This
// requires us to iterate over each list and flatten.
func policyRuleToNonResourceAttributes(rule *rbacv1.PolicyRule) []*v1.NonResourceAttributes {
	var nonResourceAttributes []*v1.NonResourceAttributes
	for _, verb := range rule.Verbs {
		for _, nonResourceUrl := range rule.NonResourceURLs {
			nonResourceAttribute := &v1.NonResourceAttributes{
				Verb: verb,
				Path: nonResourceUrl,
			}
			nonResourceAttributes = append(nonResourceAttributes, nonResourceAttribute)
		}
	}
	return nonResourceAttributes
}
