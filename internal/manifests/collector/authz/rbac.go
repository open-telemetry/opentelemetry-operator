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

// []PolicyRule 应该包含 一个 ns list ,用ns list 来表示 是否需要创建到具体的 ns
// - 如果 ns list 非空，将在所有 ns 下，创建的 Role
// - 如果 ns list 空，将创建 ClusterRole

type DynamicRolePolicy struct {
	Namespaces []string
	Rules      []rbacv1.PolicyRule
}
