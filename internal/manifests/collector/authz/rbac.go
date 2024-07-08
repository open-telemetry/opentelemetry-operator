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

package authz

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// DynamicRolePolicy includes a namespace list to indicate whether the rules need to be created within specific namespaces
// - If the namespace list is non-empty, a Role will be created in all the specified namespaces
// - If the namespace list is empty, a ClusterRole will be created.
type DynamicRolePolicy struct {
	Namespaces []string
	Rules      []rbacv1.PolicyRule
}
