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

package v1beta1

type (
	// Mode represents how the collector should be deployed (deployment vs. daemonset)
	// +kubebuilder:validation:Enum=daemonset;deployment;sidecar;statefulset
	Mode string
)

const (
	// ModeDaemonSet specifies that the collector should be deployed as a Kubernetes DaemonSet.
	ModeDaemonSet Mode = "daemonset"

	// ModeDeployment specifies that the collector should be deployed as a Kubernetes Deployment.
	ModeDeployment Mode = "deployment"

	// ModeSidecar specifies that the collector should be deployed as a sidecar to pods.
	ModeSidecar Mode = "sidecar"

	// ModeStatefulSet specifies that the collector should be deployed as a Kubernetes StatefulSet.
	ModeStatefulSet Mode = "statefulset"
)
