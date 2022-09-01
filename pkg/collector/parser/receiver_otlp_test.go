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

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOTLPSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("otlp"))
}

func TestOTLPIsFoundByName(t *testing.T) {
	// test
	p := For(logger, "otlp", map[interface{}]interface{}{})

	// verify
	assert.Equal(t, "__otlp", p.ParserName())
}

func TestOTLPPortsOverridden(t *testing.T) {
	// prepare
	builder := NewOTLPReceiverParser(logger, "otlp", map[interface{}]interface{}{
		"protocols": map[interface{}]interface{}{
			"grpc": map[interface{}]interface{}{
				"endpoint": "0.0.0.0:1234",
			},
			"http": map[interface{}]interface{}{
				"endpoint": "0.0.0.0:1235",
			},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
	}{
		"otlp-grpc": {portNumber: 1234},
		"otlp-http": {portNumber: 1235},
	}

	t.Run("service ports overridden", func(t *testing.T) {
		// test
		ports, err := builder.Ports()

		// verify
		assert.NoError(t, err)
		assert.Len(t, ports, len(expectedResults))

		seen := map[string]bool{}
		for _, port := range ports {
			r, ok := expectedResults[port.Name]
			seen[port.Name] = true
			assert.True(t, ok, "unexpected service port %s", port.Name)
			assert.EqualValues(t, r.portNumber, port.Port)
		}
		for k := range expectedResults {
			assert.True(t, seen[k], "the port %s wasn't included in the service ports", k)
		}
	})

	t.Run("container ports overridden", func(t *testing.T) {
		// test
		ports, err := builder.ContainerPorts()

		// verify
		assert.NoError(t, err)
		assert.Len(t, ports, len(expectedResults))

		seen := map[string]bool{}
		for _, port := range ports {
			r, ok := expectedResults[port.Name]
			seen[port.Name] = true
			assert.True(t, ok, "unexpected container port %s", port.Name)
			assert.EqualValues(t, r.portNumber, port.ContainerPort)
		}
		for k := range expectedResults {
			assert.True(t, seen[k], "the port %s wasn't included in the container ports", k)
		}
	})
}

func TestOTLPExposeDefaultPorts(t *testing.T) {
	// prepare
	builder := NewOTLPReceiverParser(logger, "otlp", map[interface{}]interface{}{
		"protocols": map[interface{}]interface{}{
			"grpc": map[interface{}]interface{}{},
			"http": map[interface{}]interface{}{},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
	}{
		"otlp-grpc":        {portNumber: 4317},
		"otlp-http":        {portNumber: 4318},
		"otlp-http-legacy": {portNumber: 55681},
	}

	t.Run("service ports exposed", func(t *testing.T) {
		// test
		ports, err := builder.Ports()

		// verify
		assert.NoError(t, err)
		assert.Len(t, ports, len(expectedResults))

		seen := map[string]bool{}
		for _, port := range ports {
			r, ok := expectedResults[port.Name]
			seen[port.Name] = true
			assert.True(t, ok, "unexpected service port %s", port.Name)
			assert.EqualValues(t, r.portNumber, port.Port)
		}
		for k := range expectedResults {
			assert.True(t, seen[k], "the port %s wasn't included in the service ports", k)
		}
	})

	t.Run("container ports exposed", func(t *testing.T) {
		// test
		ports, err := builder.ContainerPorts()

		// verify
		assert.NoError(t, err)
		assert.Len(t, ports, len(expectedResults))

		seen := map[string]bool{}
		for _, port := range ports {
			r, ok := expectedResults[port.Name]
			seen[port.Name] = true
			assert.True(t, ok, "unexpected container port %s", port.Name)
			assert.EqualValues(t, r.portNumber, port.ContainerPort)
		}
		for k := range expectedResults {
			assert.True(t, seen[k], "the port %s wasn't included in the container ports", k)
		}
	})
}
