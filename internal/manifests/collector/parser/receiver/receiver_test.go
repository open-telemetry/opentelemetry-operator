// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package receiver

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var logger = logf.Log.WithName("unit-tests")

func TestReceiverPortNames(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		candidate string
		expected  string
		port      int
	}{
		{"regular case", "my-receiver", "my-receiver", 123},
		{"name too long", "long-name-long-name-long-name-long-name-long-name-long-name-long-name-long-name", "port-123", 123},
		{"name with invalid chars", "my-🦄-receiver", "port-123", 123},
		{"name starting with invalid char", "-my-receiver", "port-123", 123},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.expected, naming.PortName(tt.candidate, int32(tt.port)))
		})
	}
}

func TestIgnorekubeletstatsEndpoint(t *testing.T) {
	// ignore "kubeletstats" receiver endpoint field, this is special case
	// as this receiver gets parsed by generic receiver parser
	builder, err := For("kubeletstats", map[string]interface{}{
		"endpoint": "0.0.0.0:9000",
	})
	assert.NoError(t, err)

	// test
	ports, err := builder.Ports(logr.Discard())

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 0)
}

func TestReceiverFallbackWhenNotRegistered(t *testing.T) {
	// test
	p, err := For("myreceiver", map[string]interface{}{})
	assert.NoError(t, err)

	// test
	assert.Equal(t, "__myreceiver", p.ParserName())
}

func TestReceiverShouldFindRegisteredParser(t *testing.T) {
	// prepare
	builderCalled := false
	Register("mock", func(name string, config interface{}) (parser.ComponentPortParser, error) {
		builderCalled = true
		return &mockParser{}, nil
	})

	// test
	_, _ = For("mock", map[string]interface{}{})

	// verify
	assert.True(t, builderCalled)
}

type mockParser struct {
}

func (m *mockParser) Ports(l logr.Logger) ([]corev1.ServicePort, error) {
	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock"
}

func TestSkipPortsForScrapers(t *testing.T) {
	for r := range scraperReceivers {
		builder, err := For(r, map[string]interface{}{
			"endpoint": "0.0.0.0:42069",
		})
		assert.NoError(t, err)
		ports, err := builder.Ports(logr.Discard())
		assert.NoError(t, err)
		assert.Len(t, ports, 0)
	}
}
