// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"time"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
)

var _ operator.CollectorInstance = &standaloneCollectorInstance{}

// standaloneCollectorInstance represents a bridge-managed ConfigMap as a
// CollectorInstance. name and namespace identify the ConfigMap itself.
type standaloneCollectorInstance struct {
	name       string
	namespace  string
	createdAt  time.Time
	configBody []byte
}

func newStandaloneCollectorInstance(name, namespace string, createdAt time.Time, configBody []byte) *standaloneCollectorInstance {
	return &standaloneCollectorInstance{name: name, namespace: namespace, createdAt: createdAt, configBody: configBody}
}

func (p *standaloneCollectorInstance) GetName() string {
	return p.name
}

func (p *standaloneCollectorInstance) GetNamespace() string {
	return p.namespace
}

func (p *standaloneCollectorInstance) GetConfigMapKey() string {
	return p.GetName()
}

func (p *standaloneCollectorInstance) GetCreationTimestamp() time.Time {
	return p.createdAt
}

// GetSelectorLabels returns an empty map. Standalone mode does not report
// individual pod health, so no pod selector is needed.
func (*standaloneCollectorInstance) GetSelectorLabels() map[string]string {
	return map[string]string{}
}

// GetStatusReplicas returns an empty string. Replica status is not tracked
// at the ConfigMap level in standalone mode.
func (*standaloneCollectorInstance) GetStatusReplicas() string {
	return ""
}

func (p *standaloneCollectorInstance) GetEffectiveConfig() []byte {
	return p.configBody
}
