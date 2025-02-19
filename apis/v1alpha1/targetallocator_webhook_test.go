// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

func TestTargetAllocatorDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)

	if err := AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	tests := []struct {
		name            string
		targetallocator TargetAllocator
		expected        TargetAllocator
	}{
		{
			name:            "all fields default",
			targetallocator: TargetAllocator{},
			expected: TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &one,
					},
				},
			},
		},
		{
			name: "consistent hashing strategy",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
				},
			},
			expected: TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &one,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 1,
							},
						},
					},
					AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyConsistentHashing,
				},
			},
		},
		{
			name: "provided values in spec",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &five,
					},
				},
			},
			expected: TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &five,
					},
				},
			},
		},
		{
			name: "doesn't override unmanaged",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &five,
						ManagementState: v1beta1.ManagementStateUnmanaged,
					},
				},
			},
			expected: TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas:        &five,
						ManagementState: v1beta1.ManagementStateUnmanaged,
					},
				},
			},
		},
		{
			name: "Defined PDB",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
			expected: TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &one,
						PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{
								Type:   intstr.String,
								StrVal: "10%",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			webhook := &TargetAllocatorWebhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg: config.New(
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
			}
			ctx := context.Background()
			err := webhook.Default(ctx, &test.targetallocator)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, test.targetallocator)
		})
	}
}

func TestTargetAllocatorValidatingWebhook(t *testing.T) {
	three := int32(3)

	tests := []struct { //nolint:govet
		name             string
		targetallocator  TargetAllocator
		expectedErr      string
		expectedWarnings []string
		shouldFailSar    bool
	}{
		{
			name:            "valid empty spec",
			targetallocator: TargetAllocator{},
		},
		{
			name: "valid full spec",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Replicas: &three,
						Ports: []v1beta1.PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "port1",
									Port: 5555,
								},
							},
							{
								ServicePort: v1.ServicePort{
									Name:     "port2",
									Port:     5554,
									Protocol: v1.ProtocolUDP,
								},
							},
						},
					},
				},
			},
		},
		{
			name:          "prom CR admissions warning",
			shouldFailSar: true, // force failure
			targetallocator: TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ta",
					Namespace: "test-ns",
				},
				Spec: TargetAllocatorSpec{
					PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
						Enabled: true,
					},
				},
			},
			expectedWarnings: []string{
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - monitoring.coreos.com/servicemonitors: [*]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - monitoring.coreos.com/podmonitors: [*]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - nodes/metrics: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - services: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - endpoints: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - namespaces: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - networking.k8s.io/ingresses: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - nodes: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - pods: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - configmaps: [get]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - discovery.k8s.io/endpointslices: [get,list,watch]",
				"missing the following rules for system:serviceaccount:test-ns:test-ta-targetallocator - nonResourceURL: /metrics: [get]",
			},
		},
		{
			name:          "prom CR no admissions warning",
			shouldFailSar: false, // force SAR okay
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{},
			},
		},
		{
			name: "invalid port name",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
							{
								ServicePort: v1.ServicePort{
									// this port name contains a non alphanumeric character, which is invalid.
									Name:     "-test🦄port",
									Port:     12345,
									Protocol: v1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
		{
			name: "invalid port name, too long",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "aaaabbbbccccdddd", // len: 16, too long
									Port: 5555,
								},
							},
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
		{
			name: "invalid port num",
			targetallocator: TargetAllocator{
				Spec: TargetAllocatorSpec{
					OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
						Ports: []v1beta1.PortsSpec{
							{
								ServicePort: v1.ServicePort{
									Name: "aaaabbbbccccddd", // len: 15
									// no port set means it's 0, which is invalid
								},
							},
						},
					},
				},
			},
			expectedErr: "the OpenTelemetry Spec Ports configuration is incorrect",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			cvw := &TargetAllocatorWebhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg: config.New(
					config.WithCollectorImage("targetallocator:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
				),
				reviewer: getReviewer(test.shouldFailSar),
			}
			ctx := context.Background()
			warnings, err := cvw.ValidateCreate(ctx, &test.targetallocator)
			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expectedErr)
			}
			assert.Equal(t, len(test.expectedWarnings), len(warnings))
			assert.ElementsMatch(t, warnings, test.expectedWarnings)
		})
	}
}

func getReviewer(shouldFailSAR bool) *rbac.Reviewer {
	c := fake.NewSimpleClientset()
	c.PrependReactor("create", "subjectaccessreviews", func(action kubeTesting.Action) (handled bool, ret runtime.Object, err error) {
		// check our expectation here
		if !action.Matches("create", "subjectaccessreviews") {
			return false, nil, fmt.Errorf("must be a create for a SAR")
		}
		sar, ok := action.(kubeTesting.CreateAction).GetObject().DeepCopyObject().(*authv1.SubjectAccessReview)
		if !ok || sar == nil {
			return false, nil, fmt.Errorf("bad object")
		}
		sar.Status = authv1.SubjectAccessReviewStatus{
			Allowed: !shouldFailSAR,
			Denied:  shouldFailSAR,
		}
		return true, sar, nil
	})
	return rbac.NewReviewer(c)
}
