// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestAllowSampleAppIDsOnOpenShift(t *testing.T) {
	openShiftAnnotations := map[string]string{
		openShiftUIDRangeAnnotation:           "1001280000/10000",
		openShiftSupplementalGroupsAnnotation: "1001280000/10000",
		openShiftMCSAnnotation:                "s0:c36,c10",
	}
	openShiftPatched := map[string]string{
		openShiftUIDRangeAnnotation:           "1000/1000",
		openShiftSupplementalGroupsAnnotation: "3000/3000",
		openShiftMCSAnnotation:                "s0:c36,c10",
	}
	for _, tt := range []struct {
		name         string
		openShift    bool
		discoveryErr error
		annotations  map[string]string
		// firstGet, when set, answers the first namespace Get in place of the cluster.
		firstGet        func() (runtime.Object, error)
		timeout         time.Duration
		wantAnnotations map[string]string
		wantError       string
	}{
		{
			name:            "not OpenShift",
			annotations:     map[string]string{"other": "value"},
			wantAnnotations: map[string]string{"other": "value"},
		},
		{
			name:            "OpenShift",
			openShift:       true,
			annotations:     openShiftAnnotations,
			wantAnnotations: openShiftPatched,
		},
		{
			name:        "OpenShift annotates the namespace after the first check",
			openShift:   true,
			annotations: openShiftAnnotations,
			firstGet: func() (runtime.Object, error) {
				return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}, nil
			},
			timeout:         5 * time.Second,
			wantAnnotations: openShiftPatched,
		},
		{
			name:        "the first check fails",
			openShift:   true,
			annotations: openShiftAnnotations,
			firstGet: func() (runtime.Object, error) {
				return nil, errors.New("etcdserver: request timed out")
			},
			timeout:         5 * time.Second,
			wantAnnotations: openShiftPatched,
		},
		{
			name:      "OpenShift has not annotated the namespace",
			openShift: true,
			wantError: "wait for the openshift.io/sa.scc.mcs annotation",
		},
		{
			name:         "discovery fails",
			discoveryErr: errors.New("connection refused"),
			wantError:    "discover the OpenShift security API: connection refused",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cs := fake.NewClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns", Annotations: tt.annotations}})
			if tt.openShift {
				cs.Resources = []*metav1.APIResourceList{{GroupVersion: "security.openshift.io/v1"}}
			}
			if tt.discoveryErr != nil {
				cs.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, tt.discoveryErr
				})
			}
			if tt.firstGet != nil {
				answered := false
				cs.PrependReactor("get", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
					if answered {
						return false, nil, nil
					}
					answered = true
					obj, err := tt.firstGet()
					return true, obj, err
				})
			}
			timeout := tt.timeout
			if timeout == 0 {
				timeout = 10 * time.Millisecond
			}

			err := allowSampleAppIDsOnOpenShift(context.Background(), cs, "test-ns", timeout)

			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			ns, err := cs.CoreV1().Namespaces().Get(context.Background(), "test-ns", metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantAnnotations, ns.Annotations)
		})
	}
}
