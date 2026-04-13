// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigFile is an internal representation of an effective OpAMP config file.
type ConfigFile struct {
	Body        []byte
	ContentType string
}

// Health is an internal representation of OpAMP component health.
type Health struct {
	Healthy   bool
	Status    string
	LastError string
	StartTime time.Time
	Children  map[string]Health
}

// CollectorInstance represents a collector managed by the bridge, abstracting
// the underlying Kubernetes resource (CRD or Deployment/DaemonSet).
type CollectorInstance interface {
	GetName() string
	GetNamespace() string
	GetDeletionTimestamp() *metav1.Time
	GetConfigMap() map[string]ConfigFile
}
