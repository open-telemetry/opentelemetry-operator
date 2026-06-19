// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
)

var _ operator.CollectorInstance = &standaloneCollectorInstance{}

// standaloneCollectorInstance represents a bridge-managed standalone workload.
type standaloneCollectorInstance struct {
	name      string
	namespace string
	configMap map[string]operator.ConfigFile
}

func newStandaloneCollectorInstance(name, namespace string, configMap map[string]operator.ConfigFile) *standaloneCollectorInstance {
	return &standaloneCollectorInstance{name: name, namespace: namespace, configMap: configMap}
}

func (p *standaloneCollectorInstance) GetName() string {
	return p.name
}

func (p *standaloneCollectorInstance) GetNamespace() string {
	return p.namespace
}

func (*standaloneCollectorInstance) GetDeletionTimestamp() *metav1.Time {
	return nil
}

func (p *standaloneCollectorInstance) GetConfigMap() map[string]operator.ConfigFile {
	return p.configMap
}
