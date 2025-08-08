// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package k8s contains Kubernetes cluster feature detection.
package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
)

// Detector can detect kubernetes version.
type Detector struct {
	discoveryClient discovery.DiscoveryInterface
}

// NewDetector creates a new Kubernetes feature detector.
func NewDetector(discoveryClient discovery.DiscoveryInterface) *Detector {
	return &Detector{
		discoveryClient: discoveryClient,
	}
}

// GetKubernetesVersion returns the kubernetes version.
func (d *Detector) GetKubernetesVersion() (*version.Version, error) {
	versionInfo, err := d.discoveryClient.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	currentVersion, err := version.ParseGeneric(versionInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server version %q: %w", versionInfo.GitVersion, err)
	}

	return currentVersion, nil
}
