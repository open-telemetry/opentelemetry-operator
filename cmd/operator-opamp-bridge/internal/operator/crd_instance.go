// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var _ CollectorInstance = CRDInstance{}

// CRDInstance wraps an OpenTelemetryCollector CRD to implement CollectorInstance.
type CRDInstance struct {
	Col v1beta1.OpenTelemetryCollector
}

func newCRDInstance(col v1beta1.OpenTelemetryCollector) CRDInstance {
	return CRDInstance{Col: col}
}

func (c CRDInstance) GetName() string {
	return c.Col.GetName()
}

func (c CRDInstance) GetNamespace() string {
	return c.Col.GetNamespace()
}

func (c CRDInstance) GetDeletionTimestamp() *metav1.Time {
	return c.Col.GetDeletionTimestamp()
}

func (c CRDInstance) selectorLabels() map[string]string {
	if c.Col.Status.Scale.Selector != "" {
		selMap := map[string]string{}
		for kvPair := range strings.SplitSeq(c.Col.Status.Scale.Selector, ",") {
			kv := strings.Split(kvPair, "=")
			if len(kv) != 2 {
				continue
			}
			selMap[kv[0]] = kv[1]
		}
		return selMap
	}
	return map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", c.Col.GetNamespace(), c.Col.GetName()),
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-collector",
	}
}

func (c CRDInstance) GetConfigMap() map[string]ConfigFile {
	key := NewKubeResourceKey(c.GetNamespace(), c.GetName()).String()
	marshaled, err := yaml.Marshal(&c.Col)
	if err != nil {
		return map[string]ConfigFile{key: {}}
	}
	return map[string]ConfigFile{
		key: {
			Body:        marshaled,
			ContentType: "yaml",
		},
	}
}
