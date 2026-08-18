// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestDefaultFSGroupOnOpenShift(t *testing.T) {
	tests := []struct {
		name            string
		isOpenShift     bool
		nsAnnotations   map[string]string
		existingFSGroup *int64
		wantFSGroup     *int64
	}{
		{
			name:          "OpenShift with UID range annotation",
			isOpenShift:   true,
			nsAnnotations: map[string]string{"openshift.io/sa.scc.uid-range": "1000850000/10000"},
			wantFSGroup:   ptr.To[int64](1000850000),
		},
		{
			name:            "OpenShift with explicit fsGroup preserves it",
			isOpenShift:     true,
			nsAnnotations:   map[string]string{"openshift.io/sa.scc.uid-range": "1000850000/10000"},
			existingFSGroup: ptr.To[int64](65532),
			wantFSGroup:     ptr.To[int64](65532),
		},
		{
			name:          "non-OpenShift does not set fsGroup",
			isOpenShift:   false,
			nsAnnotations: map[string]string{"openshift.io/sa.scc.uid-range": "1000850000/10000"},
			wantFSGroup:   nil,
		},
		{
			name:          "OpenShift without annotation does not set fsGroup",
			isOpenShift:   true,
			nsAnnotations: map[string]string{},
			wantFSGroup:   nil,
		},
		{
			name:          "OpenShift with malformed annotation does not set fsGroup",
			isOpenShift:   true,
			nsAnnotations: map[string]string{"openshift.io/sa.scc.uid-range": "notanumber"},
			wantFSGroup:   nil,
		},
		{
			name:        "OpenShift with supplemental-groups annotation takes precedence",
			isOpenShift: true,
			nsAnnotations: map[string]string{
				"openshift.io/sa.scc.supplemental-groups": "1000900000/10000",
				"openshift.io/sa.scc.uid-range":           "1000850000/10000",
			},
			wantFSGroup: ptr.To[int64](1000900000),
		},
		{
			name:          "OpenShift falls back to uid-range when supplemental-groups absent",
			isOpenShift:   true,
			nsAnnotations: map[string]string{"openshift.io/sa.scc.uid-range": "1000850000/10000"},
			wantFSGroup:   ptr.To[int64](1000850000),
		},
		{
			name:          "OpenShift with dash-format annotation",
			isOpenShift:   true,
			nsAnnotations: map[string]string{"openshift.io/sa.scc.uid-range": "1000850000-1000860000"},
			wantFSGroup:   ptr.To[int64](1000850000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-ns",
					Annotations: tt.nsAnnotations,
				},
			}

			cl := fake.NewClientBuilder().
				WithScheme(reconcilerTestScheme).
				WithObjects(ns).
				Build()

			routesAvailability := openshift.RoutesNotAvailable
			if tt.isOpenShift {
				routesAvailability = openshift.RoutesAvailable
			}

			r := &OpenTelemetryCollectorReconciler{
				Client: cl,
				log:    logr.Discard(),
				config: config.Config{
					OpenShiftRoutesAvailability: routesAvailability,
				},
			}

			instance := v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "test-ns",
				},
			}
			if tt.existingFSGroup != nil {
				instance.Spec.PodSecurityContext = &corev1.PodSecurityContext{
					FSGroup: tt.existingFSGroup,
				}
			}

			ctx := context.Background()
			params, err := r.GetParams(ctx, instance)
			require.NoError(t, err)

			if tt.wantFSGroup == nil {
				if params.OtelCol.Spec.PodSecurityContext != nil {
					assert.Nil(t, params.OtelCol.Spec.PodSecurityContext.FSGroup)
				}
			} else {
				require.NotNil(t, params.OtelCol.Spec.PodSecurityContext)
				require.NotNil(t, params.OtelCol.Spec.PodSecurityContext.FSGroup)
				assert.Equal(t, *tt.wantFSGroup, *params.OtelCol.Spec.PodSecurityContext.FSGroup)
			}
		})
	}
}
