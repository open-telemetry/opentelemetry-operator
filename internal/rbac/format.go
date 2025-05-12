// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/authorization/v1"
)

// WarningsGroupedByResource is a helper to take the missing permissions and format them as warnings.
func WarningsGroupedByResource(reviews []*v1.SubjectAccessReview) []string {
	userFullResourceToVerbs := make(map[string]map[string][]string)
	for _, review := range reviews {
		if _, ok := userFullResourceToVerbs[review.Spec.User]; !ok {
			userFullResourceToVerbs[review.Spec.User] = make(map[string][]string)
		}
		if review.Spec.ResourceAttributes != nil {
			key := fmt.Sprintf("%s/%s", review.Spec.ResourceAttributes.Group, review.Spec.ResourceAttributes.Resource)
			if len(review.Spec.ResourceAttributes.Group) == 0 {
				key = review.Spec.ResourceAttributes.Resource
			}
			userFullResourceToVerbs[review.Spec.User][key] = append(userFullResourceToVerbs[review.Spec.User][key], review.Spec.ResourceAttributes.Verb)
		} else if review.Spec.NonResourceAttributes != nil {
			key := fmt.Sprintf("nonResourceURL: %s", review.Spec.NonResourceAttributes.Path)
			userFullResourceToVerbs[review.Spec.User][key] = append(userFullResourceToVerbs[review.Spec.User][key], review.Spec.NonResourceAttributes.Verb)
		}
	}
	var warnings []string
	for user, fullResourceToVerbs := range userFullResourceToVerbs {
		for fullResource, verbs := range fullResourceToVerbs {
			warnings = append(warnings, fmt.Sprintf("missing the following rules for %s - %s: [%s]", user, fullResource, strings.Join(verbs, ",")))
		}
	}
	return warnings
}
