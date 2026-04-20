// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestServicePorts(t *testing.T) {
	targetAllocator := targetAllocatorInstance()
	cfg := config.New()

	params := Params{
		TargetAllocator: targetAllocator,
		Config:          cfg,
		Log:             logger,
	}

	ports := []v1.ServicePort{{Name: "targetallocation", Port: 80, TargetPort: intstr.FromString("http")}}

	s := Service(params)

	assert.Equal(t, ports[0].Name, s.Spec.Ports[0].Name)
	assert.Equal(t, ports[0].Port, s.Spec.Ports[0].Port)
	assert.Equal(t, ports[0].TargetPort, s.Spec.Ports[0].TargetPort)
}

func TestServicePortsWithTargetAllocatorMTLS(t *testing.T) {
	targetAllocator := targetAllocatorInstance()
	collector := collectorInstance()
	collector.Spec.TargetAllocator.Enabled = true
	collector.Spec.TargetAllocator.Mtls = &v1beta1.TargetAllocatorMTLS{Enabled: true}
	cfg := config.Config{
		CertManagerAvailability: certmanager.Available,
	}

	params := Params{
		Collector:       collector,
		TargetAllocator: targetAllocator,
		Config:          cfg,
		Log:             logger,
	}

	ports := []v1.ServicePort{
		{Name: "targetallocation", Port: 80, TargetPort: intstr.FromString("http")},
		{Name: "targetallocation-https", Port: 443, TargetPort: intstr.FromString("https")},
	}

	s := Service(params)

	assert.Equal(t, ports[0].Name, s.Spec.Ports[0].Name)
	assert.Equal(t, ports[0].Port, s.Spec.Ports[0].Port)
	assert.Equal(t, ports[0].TargetPort, s.Spec.Ports[0].TargetPort)
	assert.Equal(t, ports[1].Name, s.Spec.Ports[1].Name)
	assert.Equal(t, ports[1].Port, s.Spec.Ports[1].Port)
	assert.Equal(t, ports[1].TargetPort, s.Spec.Ports[1].TargetPort)
}
