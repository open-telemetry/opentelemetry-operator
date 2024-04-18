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

package receivers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestJaegerSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("jaeger"))
}

func TestJaegerIsFoundByName(t *testing.T) {
	// test
	p, err := For("jaeger", map[string]interface{}{})
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "__jaeger", p.ParserName())
}

func TestJaegerMinimalConfiguration(t *testing.T) {
	// prepare
	builder, err := For("jaeger", map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{},
		},
	})
	assert.NoError(t, err)

	// test
	ports, err := builder.Ports(logr.Discard())

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 14250, ports[0].Port)
	assert.EqualValues(t, corev1.ProtocolTCP, ports[0].Protocol)
}

func TestJaegerPortsOverridden(t *testing.T) {
	// prepare
	builder, err := For("jaeger", map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{
				"endpoint": "0.0.0.0:1234",
			},
		},
	})
	assert.NoError(t, err)

	// test
	ports, err := builder.Ports(logr.Discard())

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 1)
	assert.EqualValues(t, 1234, ports[0].Port)
	assert.EqualValues(t, corev1.ProtocolTCP, ports[0].Protocol)
}

func TestJaegerExposeDefaultPorts(t *testing.T) {
	// prepare
	builder, err := For("jaeger", map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc":           map[string]interface{}{},
			"thrift_http":    map[string]interface{}{},
			"thrift_compact": map[string]interface{}{},
			"thrift_binary":  map[string]interface{}{},
		},
	})
	assert.NoError(t, err)

	expectedResults := map[string]struct {
		transportProtocol corev1.Protocol
		portNumber        int32
		seen              bool
	}{
		"jaeger-grpc": {portNumber: 14250, transportProtocol: corev1.ProtocolTCP},
		"port-14268":  {portNumber: 14268, transportProtocol: corev1.ProtocolTCP},
		"port-6831":   {portNumber: 6831, transportProtocol: corev1.ProtocolUDP},
		"port-6832":   {portNumber: 6832, transportProtocol: corev1.ProtocolUDP},
	}

	// test
	ports, err := builder.Ports(logr.Discard())

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, 4)

	for _, port := range ports {
		r := expectedResults[port.Name]
		r.seen = true
		expectedResults[port.Name] = r
		assert.EqualValues(t, r.portNumber, port.Port)
		assert.EqualValues(t, r.transportProtocol, port.Protocol)
	}
	for k, v := range expectedResults {
		assert.True(t, v.seen, "the port %s wasn't included in the service ports", k)
	}
}
