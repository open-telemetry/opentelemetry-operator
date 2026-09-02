// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PermissionReviewClient interface {
	Create(context.Context, client.Object, ...client.CreateOption) error
}

type Permission struct {
	Verb      string
	APIGroup  string
	Resource  string
	Namespace string
	Name      string
}

// CheckPermissions verifies every requested Kubernetes permission using SelfSubjectAccessReview.
// perms describes the verb, resource, namespace, and optional object name that the bridge must be allowed to access.
func CheckPermissions(ctx context.Context, k8sClient PermissionReviewClient, perms []Permission) error {
	if len(perms) == 0 {
		return nil
	}
	if k8sClient == nil {
		return errors.New("permission review client is required")
	}
	var errs []error
	for _, perm := range perms {
		if err := checkPermission(ctx, k8sClient, perm); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// checkPermission checks if the process has the given permission. It returns an error if the perm
// is missing otherwise returns nil.
func checkPermission(ctx context.Context, k8sClient PermissionReviewClient, perm Permission) error {
	review := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:      perm.Verb,
				Group:     perm.APIGroup,
				Resource:  perm.Resource,
				Namespace: perm.Namespace,
				Name:      perm.Name,
			},
		},
	}
	if err := k8sClient.Create(ctx, review); err != nil {
		return fmt.Errorf("failed to check %s permission for %s: %w", perm.Verb, perm.description(), err)
	}
	if review.Status.Allowed {
		return nil
	}
	reason := review.Status.Reason
	if review.Status.EvaluationError != "" {
		reason = strings.TrimSpace(reason + ": " + review.Status.EvaluationError)
	}
	if reason == "" {
		reason = "access denied"
	}
	return fmt.Errorf("missing %s permission for %s: %s", perm.Verb, perm.description(), reason)
}

func (p Permission) description() string {
	resource := p.Resource
	if p.APIGroup != "" {
		resource = p.APIGroup + "/" + resource
	}
	if p.Namespace == "" && p.Name == "" {
		return resource
	}
	if p.Name == "" {
		return fmt.Sprintf("%s in namespace %s", resource, p.Namespace)
	}
	return fmt.Sprintf("%s %s/%s", resource, p.Namespace, p.Name)
}
