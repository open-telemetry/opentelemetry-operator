// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var _ CollectorInstance = CRDInstance{}

// CRDInstance wraps an OpenTelemetryCollector CRD to implement CollectorInstance.
type CRDInstance struct {
	Col v1beta1.OpenTelemetryCollector
}

func (c CRDInstance) GetName() string {
	return c.Col.GetName()
}

func (c CRDInstance) GetNamespace() string {
	return c.Col.GetNamespace()
}

func (c CRDInstance) GetCreationTimestamp() time.Time {
	return c.Col.GetCreationTimestamp().Time
}

func (c CRDInstance) GetSelectorLabels() map[string]string {
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

func (c CRDInstance) GetStatusReplicas() string {
	return c.Col.Status.Scale.StatusReplicas
}

func (c CRDInstance) GetEffectiveConfig() []byte {
	marshaled, err := yaml.Marshal(&c.Col)
	if err != nil {
		return nil
	}
	return marshaled
}
