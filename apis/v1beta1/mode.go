// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
