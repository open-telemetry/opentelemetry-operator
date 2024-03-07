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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSyslogSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("syslog"))
}

func TestSyslogIsFoundByName(t *testing.T) {
	// test
	p, err := For(logger, "syslog", map[interface{}]interface{}{})
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "__syslog", p.ParserName())
}

func TestSyslogConfiguration(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		config   map[interface{}]interface{}
		expected []corev1.ServicePort
	}{
		{"Empty configuration", map[interface{}]interface{}{}, []corev1.ServicePort{}},
		{"UDP port configuration",
			map[interface{}]interface{}{"udp": map[interface{}]interface{}{"listen_address": "0.0.0.0:1234"}},
			[]corev1.ServicePort{{Name: "syslog", Port: 1234, Protocol: corev1.ProtocolUDP}}},
		{"TCP port configuration",
			map[interface{}]interface{}{"tcp": map[interface{}]interface{}{"listen_address": "0.0.0.0:1234"}},
			[]corev1.ServicePort{{Name: "syslog", Port: 1234, Protocol: corev1.ProtocolTCP}}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			builder := NewSyslogReceiverParser(logger, "syslog", tt.config)

			// test
			ports, err := builder.Ports()

			// verify
			assert.NoError(t, err)
			assert.Equal(t, ports, tt.expected)
		})
	}
}
