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
)

func TestTcpLogSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("tcplog"))
}

func TestTcpLogIsFoundByName(t *testing.T) {
	// test
	p, err := For("tcplog", map[string]interface{}{})
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "__tcplog", p.ParserName())
}

func TestTcpLogConfiguration(t *testing.T) {
	for _, tt := range []struct {
		desc        string
		config      map[string]interface{}
		expected    []corev1.ServicePort
		expectedErr bool
	}{
		{"Empty configuration", map[string]interface{}{}, []corev1.ServicePort{}, true},
		{"TCP port configuration",
			map[string]interface{}{"listen_address": "0.0.0.0:1234"},
			[]corev1.ServicePort{{Name: "tcplog", Port: 1234, Protocol: corev1.ProtocolTCP}},
			false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			builder, err := For("tcplog", tt.config)
			assert.NoError(t, err)

			// test
			ports, err := builder.Ports(logr.Discard())

			// verify
			if tt.expectedErr {
				assert.Error(t, err, "expecting an error")
			} else {
				assert.NoError(t, err, "not expecting an error")
			}
			assert.Equal(t, ports, tt.expected)
		})
	}
}
