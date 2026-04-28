// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestResolveInitContainerSecurityContext(t *testing.T) {
	runAsNonRoot := true
	readOnlyRootFS := true
	specCtx := &corev1.SecurityContext{
		RunAsNonRoot:           &runAsNonRoot,
		ReadOnlyRootFilesystem: &readOnlyRootFS,
	}

	privileged := true
	containerCtx := &corev1.SecurityContext{
		Privileged: &privileged,
	}

	tests := []struct {
		name      string
		spec      *corev1.SecurityContext
		container *corev1.SecurityContext
		want      *corev1.SecurityContext
	}{
		{
			name:      "spec override wins when set",
			spec:      specCtx,
			container: containerCtx,
			want:      specCtx,
		},
		{
			name:      "fall back to container when spec is nil",
			spec:      nil,
			container: containerCtx,
			want:      containerCtx,
		},
		{
			name:      "both nil returns nil",
			spec:      nil,
			container: nil,
			want:      nil,
		},
		{
			name:      "spec set with container nil still uses spec",
			spec:      specCtx,
			container: nil,
			want:      specCtx,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveInitContainerSecurityContext(tt.spec, tt.container))
		})
	}
}
