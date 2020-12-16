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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

func TestReceiverPortNames(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		candidate string
		port      int
		expected  string
	}{
		{"regular case", "my-receiver", 123, "my-receiver"},
		{"name too long", "long-name-long-name-long-name-long-name-long-name-long-name-long-name-long-name", 123, "port-123"},
		{"name with invalid chars", "my-🦄-receiver", 123, "port-123"},
		{"name starting with invalid char", "-my-receiver", 123, "port-123"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.expected, portName(tt.candidate, int32(tt.port)))
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

func TestReceiverFallbackWhenNotRegistered(t *testing.T) {
	// test
	p := For(logger, "myreceiver", map[interface{}]interface{}{})

	// test
	assert.Equal(t, "__generic", p.ParserName())
}

func TestReceiverShouldFindRegisteredParser(t *testing.T) {
	// prepare
	builderCalled := false
	Register("mock", func(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
		builderCalled = true
		return &mockParser{}
	})

	// test
	For(logger, "mock", map[interface{}]interface{}{})

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
