// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestServiceAccountNewDefault(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	// test
	sa := ServiceAccountName(otelcol)

	// verify
	assert.Equal(t, "my-instance-collector", sa)
}

func TestServiceAccountOverride(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ServiceAccount: "my-special-sa",
			},
		},
	}

	// test
	sa := ServiceAccountName(otelcol)

	// verify
	assert.Equal(t, "my-special-sa", sa)
}
