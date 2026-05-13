// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"time"
)

// CollectorInstance represents a collector managed by the bridge, abstracting
// the underlying Kubernetes resource (CRD or Deployment/DaemonSet).
type CollectorInstance interface {
	GetName() string
	GetNamespace() string
	GetConfigMapKey() string
	GetCreationTimestamp() time.Time
	GetSelectorLabels() map[string]string
	GetStatusReplicas() string
	GetEffectiveConfig() []byte
}
