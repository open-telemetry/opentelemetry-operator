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

func TestReceiverType(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		name     string
		expected string
	}{
		{"regular case", "myreceiver", "myreceiver"},
		{"named instance", "myreceiver/custom", "myreceiver"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test and verify
			assert.Equal(t, tt.expected, receiverType(tt.name))
		})
	}
}

func TestReceiverParsePortFromEndpoint(t *testing.T) {
	for _, tt := range []struct {
		desc          string
		endpoint      string
		expected      int
		errorExpected bool
	}{
		{"regular case", "http://localhost:1234", 1234, false},
		{"absolute with path", "http://localhost:1234/server-status?auto", 1234, false},
		{"no protocol", "0.0.0.0:1234", 1234, false},
		{"just port", ":1234", 1234, false},
		{"no port at all", "http://localhost", 0, true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test
			val, err := portFromEndpoint(tt.endpoint)
			if tt.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.EqualValues(t, tt.expected, val, "wrong port from endpoint %s: %d", tt.endpoint, val)
		})
	}
}

func TestReceiverFailsWhenPortIsntString(t *testing.T) {
	// prepare
	config := map[interface{}]interface{}{
		"endpoint": 123,
	}

	// test
	p := singlePortFromConfigEndpoint(logger, "myreceiver", config)

	// verify
	assert.Nil(t, p)
}

func TestIgnorekubeletstatsEndpoint(t *testing.T) {
	// ignore "kubeletstats" receiver endpoint field, this is special case
	// as this receiver gets parsed by generic receiver parser
	builder := NewGenericReceiverParser(logger, "kubeletstats", map[interface{}]interface{}{
		"endpoint": "0.0.0.0:9000",
	})

	// test
	ports, err := builder.Ports()

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 0)
}

func TestReceiverFallbackWhenNotRegistered(t *testing.T) {
	// test
	p, err := For(logger, "myreceiver", map[interface{}]interface{}{})
	assert.NoError(t, err)

	// test
	assert.Equal(t, "__generic", p.ParserName())
}

func TestReceiverShouldFindRegisteredParser(t *testing.T) {
	// prepare
	builderCalled := false
	Register("mock", func(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ComponentPortParser {
		builderCalled = true
		return &mockParser{}
	})

	// test
	_, _ = For(logger, "mock", map[interface{}]interface{}{})

	// verify
	assert.True(t, builderCalled)
}

type mockParser struct {
}

func (m *mockParser) Ports() ([]corev1.ServicePort, error) {
	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock"
}

func TestSkipPortsForScrapers(t *testing.T) {
	for receiver := range scraperReceivers {
		builder := NewGenericReceiverParser(logger, receiver, map[interface{}]interface{}{
			"endpoint": "0.0.0.0:42069",
		})
		ports, err := builder.Ports()
		assert.NoError(t, err)
		assert.Len(t, ports, 0)
	}
}
