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
