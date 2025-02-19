// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyV1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	runtimecluster "sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	baseTaImage    = "something:tag"
	updatedTaImage = "another:tag"
	expectHostname = "something-else.com"
	labelName      = "something"
	labelVal       = "great"
	annotationName = "io.opentelemetry/test"
	annotationVal  = "true"
)

var (
	extraPorts = v1alpha1.PortsSpec{
		ServicePort: v1.ServicePort{
			Name:       "port-web",
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.FromInt32(8080),
		},
	}
)

type check[T any] func(t *testing.T, params T)

func TestOpenTelemetryCollectorReconciler_Reconcile(t *testing.T) {
	// enable the collector CR feature flag, as these tests assume it
	// TODO: drop this after the flag is enabled by default
	registry := colfeaturegate.GlobalRegistry()
	current := featuregate.CollectorUsesTargetAllocatorCR.IsEnabled()
	require.False(t, current, "don't set gates which are enabled by default")
	regErr := registry.Set(featuregate.CollectorUsesTargetAllocatorCR.ID(), true)
	require.NoError(t, regErr)
	t.Cleanup(func() {
		err := registry.Set(featuregate.CollectorUsesTargetAllocatorCR.ID(), current)
		require.NoError(t, err)
	})

	addedMetadataDeployment := testCollectorWithMode("test-deployment", v1alpha1.ModeDeployment)
	addedMetadataDeployment.Labels = map[string]string{
		labelName: labelVal,
	}
	addedMetadataDeployment.Annotations = map[string]string{
		annotationName: annotationVal,
	}
	deploymentExtraPorts := testCollectorWithModeAndReplicas("test-deployment", v1alpha1.ModeDeployment, 3)
	deploymentExtraPorts.Spec.Ports = append(deploymentExtraPorts.Spec.Ports, extraPorts)
	deploymentExtraPorts.Spec.DeploymentUpdateStrategy = appsv1.DeploymentStrategy{
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			MaxSurge: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}
	deploymentExtraPorts.Annotations = map[string]string{
		"new-annotation": "new-value",
	}
	baseOTLPParams := testCollectorAssertNoErr(t, "test-otlp", "", otlpTestFile)
	ingressParams := testCollectorAssertNoErr(t, "test-ingress", "", testFileIngress)
	ingressParams.Spec.Ingress.Type = "ingress"
	updatedIngressParams := testCollectorAssertNoErr(t, "test-ingress", "", testFileIngress)
	updatedIngressParams.Spec.Ingress.Type = "ingress"
	updatedIngressParams.Spec.Ingress.Annotations = map[string]string{"blub": "blob"}
	updatedIngressParams.Spec.Ingress.Hostname = expectHostname
	routeParams := testCollectorAssertNoErr(t, "test-route", "", testFileIngress)
	routeParams.Spec.Ingress.Type = v1alpha1.IngressTypeRoute
	routeParams.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeInsecure
	updatedRouteParams := testCollectorAssertNoErr(t, "test-route", "", testFileIngress)
	updatedRouteParams.Spec.Ingress.Type = v1alpha1.IngressTypeRoute
	updatedRouteParams.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeInsecure
	updatedRouteParams.Spec.Ingress.Hostname = expectHostname
	deletedParams := testCollectorWithMode("test2", v1alpha1.ModeDeployment)
	now := metav1.NewTime(time.Now())
	deletedParams.DeletionTimestamp = &now

	type args struct {
		params v1alpha1.OpenTelemetryCollector
		// an optional list of updates to supply after the initial object
		updates []v1alpha1.OpenTelemetryCollector
	}
	type want struct {
		// result check
		result controllerruntime.Result
		// a check to run against the current state applied
		checks []check[v1alpha1.OpenTelemetryCollector]
		// if an error from creation validation is expected
		validateErr assert.ErrorAssertionFunc
		// if an error from reconciliation is expected
		wantErr assert.ErrorAssertionFunc
	}
	tests := []struct {
		name string
		args args
		want []want
	}{
		{
			name: "deployment collector",
			args: args{
				params:  addedMetadataDeployment,
				updates: []v1alpha1.OpenTelemetryCollector{deploymentExtraPorts},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Collector(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(2), *d.Spec.Replicas)
							assert.Contains(t, d.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							// confirm the initial strategy is unset
							assert.Equal(t, d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal, int32(0))
							assert.Equal(t, d.Spec.Strategy.RollingUpdate.MaxSurge.IntVal, int32(0))
							svc := &v1.Service{}
							exists, err = populateObjectIfExists(t, svc, namespacedObjectName(naming.Service(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, svc.Spec.Selector, map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.test-deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							})
							sa := &v1.ServiceAccount{}
							exists, err = populateObjectIfExists(t, sa, namespacedObjectName(naming.ServiceAccount(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, map[string]string{
								annotationName: "true",
							}, sa.Annotations)
							saPatch := sa.DeepCopy()
							saPatch.Annotations["user-defined-annotation"] = "value"
							err = k8sClient.Patch(ctx, saPatch, client.MergeFrom(sa))
							require.NoError(t, err)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Collector(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(3), *d.Spec.Replicas)
							// confirm the strategy has been changed
							assert.Equal(t, d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal, int32(1))
							assert.Equal(t, d.Spec.Strategy.RollingUpdate.MaxSurge.IntVal, int32(1))
							// confirm that we don't remove annotations and labels even if we don't set them
							assert.Contains(t, d.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							actual := v1.Service{}
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.Service(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Contains(t, actual.Spec.Ports, extraPorts.ServicePort)
							assert.Equal(t, actual.Spec.Selector, map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.test-deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							})

							sa := &v1.ServiceAccount{}
							exists, err = populateObjectIfExists(t, sa, namespacedObjectName(naming.ServiceAccount(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, map[string]string{
								annotationName:            "true",
								"user-defined-annotation": "value",
								"new-annotation":          "new-value",
							}, sa.Annotations)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},

		{
			name: "otlp receiver collector",
			args: args{
				params:  baseOTLPParams,
				updates: []v1alpha1.OpenTelemetryCollector{},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							d := appsv1.StatefulSet{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Collector(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(1), *d.Spec.Replicas)
							svc := &v1.Service{}
							exists, err = populateObjectIfExists(t, svc, namespacedObjectName(naming.Service(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, svc.Spec.Selector, map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.test-otlp",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							})
							assert.Len(t, svc.Spec.Ports, 4)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "invalid mode",
			args: args{
				params:  testCollectorWithMode("test-invalid", "bad"),
				updates: []v1alpha1.OpenTelemetryCollector{},
			},
			want: []want{
				{
					result:  controllerruntime.Result{},
					checks:  []check[v1alpha1.OpenTelemetryCollector]{},
					wantErr: assert.NoError,
					validateErr: func(t assert.TestingT, err2 error, msgAndArgs ...interface{}) bool {
						return assert.ErrorContains(t, err2, "Unsupported value: \"bad\"", msgAndArgs)
					},
				},
			},
		},
		{
			name: "invalid prometheus configuration",
			args: args{
				params:  testCollectorAssertNoErr(t, "test-invalid-prom", baseTaImage, testFileIngress),
				updates: []v1alpha1.OpenTelemetryCollector{},
			},
			want: []want{
				{
					result:  controllerruntime.Result{},
					checks:  []check[v1alpha1.OpenTelemetryCollector]{},
					wantErr: assert.NoError,
					validateErr: func(t assert.TestingT, err2 error, msgAndArgs ...interface{}) bool {
						return assert.ErrorContains(t, err2, "no prometheus available as part of the configuration", msgAndArgs)
					},
				},
			},
		},
		{
			name: "deployment collector with ingress",
			args: args{
				params:  ingressParams,
				updates: []v1alpha1.OpenTelemetryCollector{updatedIngressParams},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							d := networkingv1.Ingress{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Ingress(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							d := networkingv1.Ingress{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Ingress(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, "something-else.com", d.Spec.Rules[0].Host)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "deployment collector with routes",
			args: args{
				params:  routeParams,
				updates: []v1alpha1.OpenTelemetryCollector{updatedRouteParams},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							got := routev1.Route{}
							nsn := types.NamespacedName{Namespace: params.Namespace, Name: "otlp-grpc-test-route-route"}
							exists, err := populateObjectIfExists(t, &got, nsn)
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							got := routev1.Route{}
							nsn := types.NamespacedName{Namespace: params.Namespace, Name: "otlp-grpc-test-route-route"}
							exists, err := populateObjectIfExists(t, &got, nsn)
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, "otlp-grpc.something-else.com", got.Spec.Host)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "hpa v2 deployment collector",
			args: args{
				params:  testCollectorWithHPA(3, 5),
				updates: []v1alpha1.OpenTelemetryCollector{testCollectorWithHPA(1, 9)},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := autoscalingv2.HorizontalPodAutoscaler{}
							exists, hpaErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.Name), params.Namespace))
							assert.NoError(t, hpaErr)
							require.Len(t, actual.Spec.Metrics, 1)
							assert.Equal(t, int32(90), *actual.Spec.Metrics[0].Resource.Target.AverageUtilization)
							assert.Equal(t, int32(3), *actual.Spec.MinReplicas)
							assert.Equal(t, int32(5), actual.Spec.MaxReplicas)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := autoscalingv2.HorizontalPodAutoscaler{}
							exists, hpaErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.Name), params.Namespace))
							assert.NoError(t, hpaErr)
							require.Len(t, actual.Spec.Metrics, 1)
							assert.Equal(t, int32(90), *actual.Spec.Metrics[0].Resource.Target.AverageUtilization)
							assert.Equal(t, int32(1), *actual.Spec.MinReplicas)
							assert.Equal(t, int32(9), actual.Spec.MaxReplicas)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "policy v1 deployment collector",
			args: args{
				params:  testCollectorWithModeAndReplicas("policytest", v1alpha1.ModeDeployment, 3),
				updates: []v1alpha1.OpenTelemetryCollector{testCollectorWithPDB(1, 0)},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := policyV1.PodDisruptionBudget{}
							exists, pdbErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.Name), params.Namespace))
							assert.NoError(t, pdbErr)
							assert.Equal(t, int32(1), actual.Spec.MaxUnavailable.IntVal)
							assert.Nil(t, actual.Spec.MinAvailable)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := policyV1.PodDisruptionBudget{}
							exists, pdbErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.Name), params.Namespace))
							assert.NoError(t, pdbErr)
							assert.Nil(t, actual.Spec.MaxUnavailable)
							assert.Equal(t, int32(1), actual.Spec.MinAvailable.IntVal)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "daemonset collector",
			args: args{
				params: testCollectorWithMode("test-daemonset", v1alpha1.ModeDaemonSet),
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							exists, err := populateObjectIfExists(t, &appsv1.DaemonSet{}, namespacedObjectName(naming.Collector(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "stateful should update collector with TA",
			args: args{
				params: testCollectorWithMode("test-stateful-ta", v1alpha1.ModeStatefulSet),
				updates: []v1alpha1.OpenTelemetryCollector{
					testCollectorAssertNoErr(t, "test-stateful-ta", baseTaImage, promFile),
					testCollectorAssertNoErr(t, "test-stateful-ta", baseTaImage, updatedPromFile),
					testCollectorAssertNoErr(t, "test-stateful-ta", updatedTaImage, updatedPromFile),
				},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							configHash, _ := getConfigMapSHAFromString(params.Spec.Config)
							configHash = configHash[:8]
							exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.ConfigMap(params.Name, configHash), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &appsv1.StatefulSet{}, namespacedObjectName(naming.Collector(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							// Check the TA doesn't exist
							exists, err = populateObjectIfExists(t, &v1alpha1.TargetAllocator{}, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.False(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							configHash, _ := getConfigMapSHAFromString(params.Spec.Config)
							configHash = configHash[:8]
							exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.ConfigMap(params.Name, configHash), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							actual := v1alpha1.TargetAllocator{}
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(params.Name, params.Namespace))
							require.NoError(t, err)
							require.True(t, exists)
							expected := v1alpha1.TargetAllocator{
								ObjectMeta: metav1.ObjectMeta{
									Name:      params.Name,
									Namespace: params.Namespace,
									Labels:    nil,
								},
								Spec: v1alpha1.TargetAllocatorSpec{
									OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{},
									AllocationStrategy:        "consistent-hashing",
									FilterStrategy:            "relabel-config",
									PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
										ScrapeInterval:         &metav1.Duration{Duration: time.Second * 30},
										ServiceMonitorSelector: &metav1.LabelSelector{},
										PodMonitorSelector:     &metav1.LabelSelector{},
									},
								},
							}
							assert.Equal(t, expected.Name, actual.Name)
							assert.Equal(t, expected.Namespace, actual.Namespace)
							assert.Equal(t, expected.Labels, actual.Labels)
							assert.Equal(t, baseTaImage, actual.Spec.Image)
							assert.Equal(t, expected.Spec.AllocationStrategy, actual.Spec.AllocationStrategy)
							assert.Equal(t, expected.Spec.FilterStrategy, actual.Spec.FilterStrategy)
							assert.Equal(t, expected.Spec.ScrapeConfigs, actual.Spec.ScrapeConfigs)

						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							configHash, _ := getConfigMapSHAFromString(params.Spec.Config)
							configHash = configHash[:8]
							exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.ConfigMap(params.Name, configHash), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							actual := v1alpha1.TargetAllocator{}
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(params.Name, params.Namespace))
							require.NoError(t, err)
							require.True(t, exists)
							assert.Nil(t, actual.Spec.ScrapeConfigs)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := v1alpha1.TargetAllocator{}
							exists, err := populateObjectIfExists(t, &actual, namespacedObjectName(params.Name, params.Namespace))
							require.NoError(t, err)
							require.True(t, exists)
							assert.Equal(t, actual.Spec.Image, updatedTaImage)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
		{
			name: "collector is being deleted",
			args: args{
				params:  deletedParams,
				updates: []v1alpha1.OpenTelemetryCollector{},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							o := v1alpha1.OpenTelemetryCollector{}
							exists, err := populateObjectIfExists(t, &o, namespacedObjectName(naming.Collector(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.False(t, exists) // There should be no collector anymore
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			testContext := context.Background()
			nsn := types.NamespacedName{Name: tt.args.params.Name, Namespace: tt.args.params.Namespace}
			testCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reconciler := createTestReconciler(t, testCtx, config.New(
				config.WithCollectorImage("default-collector"),
				config.WithTargetAllocatorImage("default-ta-allocator"),
				config.WithOpenShiftRoutesAvailability(openshift.RoutesAvailable),
				config.WithPrometheusCRAvailability(prometheus.Available),
			))

			assert.True(t, len(tt.want) > 0, "must have at least one group of checks to run")
			firstCheck := tt.want[0]
			// Check for this before create, otherwise it's blown away.
			deletionTimestamp := tt.args.params.GetDeletionTimestamp()
			createErr := k8sClient.Create(testContext, &tt.args.params)
			if !firstCheck.validateErr(t, createErr) {
				return
			}
			// wait until the reconciler sees the object in its cache
			if createErr == nil {
				assert.EventuallyWithT(t, func(collect *assert.CollectT) {
					actual := &v1beta1.OpenTelemetryCollector{}
					err := reconciler.Get(testContext, nsn, actual)
					assert.NoError(collect, err)
				}, time.Second*5, time.Millisecond)
			}
			if deletionTimestamp != nil {
				err := k8sClient.Delete(testContext, &tt.args.params, client.PropagationPolicy(metav1.DeletePropagationForeground))
				assert.NoError(t, err)
				// wait until the reconciler sees the deletion
				assert.EventuallyWithT(t, func(collect *assert.CollectT) {
					actual := &v1beta1.OpenTelemetryCollector{}
					err := reconciler.Get(testContext, nsn, actual)
					assert.NoError(collect, err)
					assert.NotNil(t, actual.GetDeletionTimestamp())
				}, time.Second*5, time.Millisecond)
			}
			req := k8sreconcile.Request{
				NamespacedName: nsn,
			}
			got, reconcileErr := reconciler.Reconcile(testContext, req)
			if !firstCheck.wantErr(t, reconcileErr) {
				require.NoError(t, k8sClient.Delete(testContext, &tt.args.params))
				return
			}
			assert.Equal(t, firstCheck.result, got)
			for _, check := range firstCheck.checks {
				check(t, tt.args.params)
			}
			// run the next set of checks
			for pid, updateParam := range tt.args.updates {
				updateParam := updateParam
				existing := v1alpha1.OpenTelemetryCollector{}
				found, err := populateObjectIfExists(t, &existing, nsn)
				assert.True(t, found)
				assert.NoError(t, err)

				updateParam.SetResourceVersion(existing.ResourceVersion)
				updateParam.SetUID(existing.UID)
				err = k8sClient.Update(testContext, &updateParam)
				assert.NoError(t, err)
				if err != nil {
					continue
				}
				// wait until the reconciler sees the object in its cache
				assert.EventuallyWithT(t, func(collect *assert.CollectT) {
					actual := &v1alpha1.OpenTelemetryCollector{}
					err = reconciler.Get(testContext, nsn, actual)
					assert.NoError(collect, err)
					assert.Equal(collect, updateParam.Spec, actual.Spec)
				}, time.Second*5, time.Millisecond)
				req := k8sreconcile.Request{
					NamespacedName: nsn,
				}
				_, err = reconciler.Reconcile(testContext, req)
				// account for already checking the initial group
				checkGroup := tt.want[pid+1]
				if !checkGroup.wantErr(t, err) {
					return
				}
				assert.Equal(t, checkGroup.result, got)
				for _, check := range checkGroup.checks {
					check(t, updateParam)
				}
			}
			// Only delete upon a successful creation
			if createErr == nil {
				require.NoError(t, k8sClient.Delete(testContext, &tt.args.params))
			}
		})
	}
}

// TestOpenTelemetryCollectorReconciler_RemoveDisabled starts off with optional resources enabled, and then disables
// them one by one to ensure they're actually deleted.
func TestOpenTelemetryCollectorReconciler_RemoveDisabled(t *testing.T) {
	expectedStartingResourceCount := 11
	startingCollector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "placeholder",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			TargetAllocator: v1beta1.TargetAllocatorEmbedded{
				Enabled: true,
				PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
					Enabled: true,
				},
			},
			Mode: v1beta1.ModeStatefulSet,
			Observability: v1beta1.ObservabilitySpec{
				Metrics: v1beta1.MetricsConfigSpec{
					EnableMetrics: true,
				},
			},
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]interface{}{
							"config": map[string]interface{}{
								"scrape_configs": []interface{}{},
							},
						},
					},
				},
				Exporters: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"nop": map[string]interface{}{},
					},
				},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"logs": {
							Exporters: []string{"nop"},
							Receivers: []string{"nop"},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name                          string
		mutateCollector               func(*v1beta1.OpenTelemetryCollector)
		expectedResourcesDeletedCount int
	}{
		{
			name: "disable targetallocator",
			mutateCollector: func(obj *v1beta1.OpenTelemetryCollector) {
				obj.Spec.TargetAllocator.Enabled = false
			},
			expectedResourcesDeletedCount: 5,
		},
		{
			name: "disable metrics",
			mutateCollector: func(obj *v1beta1.OpenTelemetryCollector) {
				obj.Spec.Observability.Metrics.EnableMetrics = false
			},
			expectedResourcesDeletedCount: 1,
		},
		{
			name: "disable default service account",
			mutateCollector: func(obj *v1beta1.OpenTelemetryCollector) {
				obj.Spec.OpenTelemetryCommonFields.ServiceAccount = "placeholder"
			},
			expectedResourcesDeletedCount: 1,
		},
	}

	testCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	reconciler := createTestReconciler(t, testCtx, config.New(
		config.WithCollectorImage("default-collector"),
		config.WithTargetAllocatorImage("default-ta-allocator"),
		config.WithOpenShiftRoutesAvailability(openshift.RoutesAvailable),
		config.WithPrometheusCRAvailability(prometheus.Available),
	))

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(startingCollector.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			collectorName := sanitizeResourceName(tc.name)
			collector := startingCollector.DeepCopy()
			collector.Name = collectorName
			nsn := types.NamespacedName{Name: collector.Name, Namespace: collector.Namespace}
			clientCtx := context.Background()
			err := k8sClient.Create(clientCtx, collector)
			require.NoError(t, err)
			t.Cleanup(func() {
				deleteErr := k8sClient.Delete(clientCtx, collector)
				require.NoError(t, deleteErr)
			})
			err = k8sClient.Get(clientCtx, nsn, collector)
			require.NoError(t, err)
			req := k8sreconcile.Request{
				NamespacedName: nsn,
			}
			_, reconcileErr := reconciler.Reconcile(clientCtx, req)
			assert.NoError(t, reconcileErr)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				list, listErr := getAllOwnedResources(clientCtx, reconciler, collector, opts...)
				assert.NoError(collect, listErr)
				assert.NotEmpty(collect, list)
				assert.Len(collect, list, expectedStartingResourceCount)
			}, time.Second*5, time.Millisecond)

			err = k8sClient.Get(clientCtx, nsn, collector)
			require.NoError(t, err)
			tc.mutateCollector(collector)
			err = k8sClient.Update(clientCtx, collector)
			require.NoError(t, err)
			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				actual := &v1beta1.OpenTelemetryCollector{}
				err = reconciler.Get(clientCtx, nsn, actual)
				assert.NoError(collect, err)
				assert.Equal(collect, collector.Spec, actual.Spec)
			}, time.Second*5, time.Millisecond)

			_, reconcileErr = reconciler.Reconcile(clientCtx, req)
			assert.NoError(t, reconcileErr)

			expectedResourceCount := expectedStartingResourceCount - tc.expectedResourcesDeletedCount
			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				list, listErr := getAllOwnedResources(clientCtx, reconciler, collector, opts...)
				assert.NoError(collect, listErr)
				assert.NotEmpty(collect, list)
				assert.Len(collect, list, expectedResourceCount)
			}, time.Second*5, time.Millisecond)
		})
	}
}

func TestOpenTelemetryCollectorReconciler_VersionedConfigMaps(t *testing.T) {
	collectorName := sanitizeResourceName(t.Name())
	collector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collectorName,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{},
			},
			ConfigVersions: 1,
			TargetAllocator: v1beta1.TargetAllocatorEmbedded{
				Enabled: true,
				PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
					Enabled: true,
				},
			},
			Mode: v1beta1.ModeStatefulSet,
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]interface{}{
							"config": map[string]interface{}{
								"scrape_configs": []interface{}{},
							},
						},
						"nop": map[string]interface{}{},
					},
				},
				Exporters: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"nop": map[string]interface{}{},
					},
				},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"logs": {
							Exporters: []string{"nop"},
							Receivers: []string{"nop"},
						},
					},
				},
			},
		},
	}

	testCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	reconciler := createTestReconciler(t, testCtx, config.New(
		config.WithCollectorImage("default-collector"),
		config.WithTargetAllocatorImage("default-ta-allocator"),
		config.WithOpenShiftRoutesAvailability(openshift.RoutesAvailable),
		config.WithPrometheusCRAvailability(prometheus.Available),
	))

	nsn := types.NamespacedName{Name: collector.Name, Namespace: collector.Namespace}
	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(collector.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/instance":   naming.Truncate("%s.%s", 63, nsn.Namespace, nsn.Name),
		}),
	}

	clientCtx := context.Background()
	err := k8sClient.Create(clientCtx, collector)
	require.NoError(t, err)
	t.Cleanup(func() {
		deleteErr := k8sClient.Delete(clientCtx, collector)
		require.NoError(t, deleteErr)
	})
	err = k8sClient.Get(clientCtx, nsn, collector)
	require.NoError(t, err)
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, reconcileErr := reconciler.Reconcile(clientCtx, req)
	assert.NoError(t, reconcileErr)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		configMaps := &v1.ConfigMapList{}
		listErr := k8sClient.List(clientCtx, configMaps, opts...)
		assert.NoError(collect, listErr)
		assert.NotEmpty(collect, configMaps)
		assert.Len(collect, configMaps.Items, 2)
	}, time.Second*5, time.Millisecond)

	// modify the ConfigMap, it should be kept
	// wait a second first, as K8s creation timestamps only have second precision
	time.Sleep(time.Second)
	err = k8sClient.Get(clientCtx, nsn, collector)
	require.NoError(t, err)
	collector.Spec.Config.Exporters.Object["debug"] = map[string]interface{}{}
	err = k8sClient.Update(clientCtx, collector)
	require.NoError(t, err)
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		actual := &v1beta1.OpenTelemetryCollector{}
		err = reconciler.Get(clientCtx, nsn, actual)
		assert.NoError(collect, err)
		assert.Equal(collect, collector.Spec, actual.Spec)
	}, time.Second*5, time.Millisecond)

	_, reconcileErr = reconciler.Reconcile(clientCtx, req)
	assert.NoError(t, reconcileErr)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		configMaps := &v1.ConfigMapList{}
		listErr := k8sClient.List(clientCtx, configMaps, opts...)
		assert.NoError(collect, listErr)
		assert.NotEmpty(collect, configMaps)
		assert.Len(collect, configMaps.Items, 3)
	}, time.Second*5, time.Millisecond)

	// modify the ConfigMap again, the oldest one is still kept, but is dropped after next reconciliation
	// wait a second first, as K8s creation timestamps only have second precision
	time.Sleep(time.Second)
	err = k8sClient.Get(clientCtx, nsn, collector)
	require.NoError(t, err)
	collector.Spec.Config.Exporters.Object["debug/2"] = map[string]interface{}{}
	err = k8sClient.Update(clientCtx, collector)
	require.NoError(t, err)
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		actual := &v1beta1.OpenTelemetryCollector{}
		err = reconciler.Get(clientCtx, nsn, actual)
		assert.NoError(collect, err)
		assert.Equal(collect, collector.Spec, actual.Spec)
	}, time.Second*5, time.Millisecond)

	_, reconcileErr = reconciler.Reconcile(clientCtx, req)
	assert.NoError(t, reconcileErr)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		configMaps := &v1.ConfigMapList{}
		listErr := k8sClient.List(clientCtx, configMaps, opts...)
		assert.NoError(collect, listErr)
		assert.NotEmpty(collect, configMaps)
		assert.Len(collect, configMaps.Items, 4)
	}, time.Second*5, time.Millisecond)

	_, reconcileErr = reconciler.Reconcile(clientCtx, req)
	assert.NoError(t, reconcileErr)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		configMaps := &v1.ConfigMapList{}
		listErr := k8sClient.List(clientCtx, configMaps, opts...)
		assert.NoError(collect, listErr)
		assert.NotEmpty(collect, configMaps)
		assert.Len(collect, configMaps.Items, 3)
	}, time.Second*5, time.Second)
}

func TestOpAMPBridgeReconciler_Reconcile(t *testing.T) {
	addedMetadataDeployment := opampBridgeParams()
	addedMetadataDeployment.OpAMPBridge.Labels = map[string]string{
		labelName: labelVal,
	}
	addedMetadataDeployment.OpAMPBridge.Spec.PodAnnotations = map[string]string{
		annotationName: annotationVal,
	}
	deploymentExtraPorts := opampBridgeParams()
	deploymentExtraPorts.OpAMPBridge.Spec.Ports = append(deploymentExtraPorts.OpAMPBridge.Spec.Ports, extraPorts.ServicePort)

	type args struct {
		params manifests.Params
		// an optional list of updates to supply after the initial object
		updates []manifests.Params
	}
	type want struct {
		// result check
		result controllerruntime.Result
		// a check to run against the current state applied
		checks []check[v1alpha1.OpAMPBridge]
		// if an error from creation validation is expected
		validateErr assert.ErrorAssertionFunc
		// if an error from reconciliation is expected
		wantErr assert.ErrorAssertionFunc
	}
	tests := []struct {
		name string
		args args
		want []want
	}{
		{
			name: "deployment opamp-bridge",
			args: args{
				params:  addedMetadataDeployment,
				updates: []manifests.Params{deploymentExtraPorts},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpAMPBridge]{
						func(t *testing.T, params v1alpha1.OpAMPBridge) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.OpAMPBridge(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(1), *d.Spec.Replicas)
							assert.Contains(t, d.Spec.Template.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							exists, err = populateObjectIfExists(t, &v1.Service{}, namespacedObjectName(naming.OpAMPBridgeService(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, namespacedObjectName(naming.OpAMPBridgeServiceAccount(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpAMPBridge]{
						func(t *testing.T, params v1alpha1.OpAMPBridge) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.OpAMPBridge(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							// confirm that we don't remove annotations and labels even if we don't set them
							assert.Contains(t, d.Spec.Template.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							actual := v1.Service{}
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.OpAMPBridgeService(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Contains(t, actual.Spec.Ports, extraPorts.ServicePort)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			testContext := context.Background()
			nsn := types.NamespacedName{Name: tt.args.params.OpAMPBridge.Name, Namespace: tt.args.params.OpAMPBridge.Namespace}
			reconciler := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
				Client:   k8sClient,
				Log:      logger,
				Scheme:   testScheme,
				Recorder: record.NewFakeRecorder(20),
				Config: config.New(
					config.WithCollectorImage("default-collector"),
					config.WithTargetAllocatorImage("default-ta-allocator"),
					config.WithOperatorOpAMPBridgeImage("default-opamp-bridge"),
				),
			})
			assert.True(t, len(tt.want) > 0, "must have at least one group of checks to run")
			firstCheck := tt.want[0]
			createErr := k8sClient.Create(testContext, &tt.args.params.OpAMPBridge)
			if !firstCheck.validateErr(t, createErr) {
				return
			}
			req := k8sreconcile.Request{
				NamespacedName: nsn,
			}
			got, reconcileErr := reconciler.Reconcile(testContext, req)
			if !firstCheck.wantErr(t, reconcileErr) {
				require.NoError(t, k8sClient.Delete(testContext, &tt.args.params.OpAMPBridge))
				return
			}
			assert.Equal(t, firstCheck.result, got)
			for _, check := range firstCheck.checks {
				check(t, tt.args.params.OpAMPBridge)
			}
			// run the next set of checks
			for pid, updateParam := range tt.args.updates {
				updateParam := updateParam
				existing := v1alpha1.OpAMPBridge{}
				found, err := populateObjectIfExists(t, &existing, nsn)
				assert.True(t, found)
				assert.NoError(t, err)

				updateParam.OpAMPBridge.SetResourceVersion(existing.ResourceVersion)
				updateParam.OpAMPBridge.SetUID(existing.UID)
				err = k8sClient.Update(testContext, &updateParam.OpAMPBridge)
				assert.NoError(t, err)
				if err != nil {
					continue
				}
				req := k8sreconcile.Request{
					NamespacedName: nsn,
				}
				_, err = reconciler.Reconcile(testContext, req)
				// account for already checking the initial group
				checkGroup := tt.want[pid+1]
				if !checkGroup.wantErr(t, err) {
					return
				}
				assert.Equal(t, checkGroup.result, got)
				for _, check := range checkGroup.checks {
					check(t, updateParam.OpAMPBridge)
				}
			}
			// Only delete upon a successful creation
			if createErr == nil {
				require.NoError(t, k8sClient.Delete(testContext, &tt.args.params.OpAMPBridge))
			}
		})
	}
}

func TestSkipWhenInstanceDoesNotExist(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := controllers.NewReconciler(controllers.Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
	})

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err := reconciler.Reconcile(context.Background(), req)

	// verify
	assert.NoError(t, err)
}

func TestRegisterWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)

	reconciler := controllers.NewReconciler(controllers.Params{})

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	assert.NoError(t, err)
}

func TestOpenTelemetryCollectorReconciler_Finalizer(t *testing.T) {
	otelcol := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "otel-k8sattrs",
			Namespace: "test-finalizer",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode: v1alpha1.ModeDeployment,
			Config: `
processors:
  k8sattributes:
receivers:
  otlp:
    protocols:
      grpc:

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [k8sattributes]
      exporters: [debug]
`,
		},
	}

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: otelcol.Namespace,
		},
	}
	clientErr := k8sClient.Create(context.Background(), ns)
	require.NoError(t, clientErr)
	clientErr = k8sClient.Create(context.Background(), otelcol)
	require.NoError(t, clientErr)

	testCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reconciler := createTestReconciler(t, testCtx, config.New(
		config.WithCollectorImage("default-collector"),
		config.WithTargetAllocatorImage("default-ta-allocator"),
		config.WithRBACPermissions(autoRBAC.Available),
	))

	nsn := types.NamespacedName{Name: otelcol.Name, Namespace: otelcol.Namespace}
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	reconcile, reconcileErr := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, reconcileErr)
	require.False(t, reconcile.Requeue)

	colClusterRole := &rbacv1.ClusterRole{}
	clientErr = k8sClient.Get(context.Background(), types.NamespacedName{
		Name: naming.ClusterRole(otelcol.Name, otelcol.Namespace),
	}, colClusterRole)
	require.NoError(t, clientErr)
	colClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	clientErr = k8sClient.Get(context.Background(), types.NamespacedName{
		Name: naming.ClusterRoleBinding(otelcol.Name, otelcol.Namespace),
	}, colClusterRoleBinding)
	require.NoError(t, clientErr)

	// delete collector and check if the cluster role was deleted
	clientErr = k8sClient.Delete(context.Background(), otelcol)
	require.NoError(t, clientErr)
	// wait until the reconciler sees the object as deleted in its cache
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		actual := &v1beta1.OpenTelemetryCollector{}
		err := reconciler.Get(context.Background(), nsn, actual)
		assert.NoError(collect, err)
		assert.NotNil(t, actual.GetDeletionTimestamp())
	}, time.Second*5, time.Millisecond)

	reconcile, reconcileErr = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, reconcileErr)
	require.False(t, reconcile.Requeue)

	clientErr = k8sClient.Get(context.Background(), types.NamespacedName{
		Name: naming.ClusterRole(otelcol.Name, otelcol.Namespace),
	}, colClusterRole)
	require.Error(t, clientErr)
	clientErr = k8sClient.Get(context.Background(), types.NamespacedName{
		Name: naming.ClusterRoleBinding(otelcol.Name, otelcol.Namespace),
	}, colClusterRoleBinding)
	require.Error(t, clientErr)
}

func namespacedObjectName(name string, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

// getAllResources gets all the resource types owned by the controller.
func getAllOwnedResources(
	ctx context.Context,
	reconciler *controllers.OpenTelemetryCollectorReconciler,
	owner *v1beta1.OpenTelemetryCollector,
	options ...client.ListOption,
) ([]client.Object, error) {
	ownedResourceTypes := reconciler.GetOwnedResourceTypes()
	allResources := []client.Object{}
	for _, resourceType := range ownedResourceTypes {
		list := &unstructured.UnstructuredList{}
		gvk, err := apiutil.GVKForObject(resourceType, k8sClient.Scheme())
		if err != nil {
			return nil, err
		}
		list.SetGroupVersionKind(gvk)
		err = k8sClient.List(ctx, list, options...)
		if err != nil {
			return []client.Object{}, fmt.Errorf("error listing %s: %w", gvk.Kind, err)
		}
		for _, obj := range list.Items {
			if obj.GetDeletionTimestamp() != nil {
				continue
			}

			newObj := obj
			if !IsOwnedBy(&newObj, owner) {
				continue
			}
			allResources = append(allResources, &newObj)
		}
	}
	return allResources, nil
}

func IsOwnedBy(obj metav1.Object, owner *v1beta1.OpenTelemetryCollector) bool {
	if obj.GetNamespace() != owner.GetNamespace() {
		labels := obj.GetLabels()
		instanceLabelValue := labels["app.kubernetes.io/instance"]
		return instanceLabelValue == naming.Truncate("%s.%s", 63, owner.Namespace, owner.Name)
	}
	ownerReferences := obj.GetOwnerReferences()
	isOwner := slices.ContainsFunc(ownerReferences, func(ref metav1.OwnerReference) bool {
		return ref.UID == owner.GetUID()
	})
	return isOwner
}

func createTestReconciler(t *testing.T, ctx context.Context, cfg config.Config) *controllers.OpenTelemetryCollectorReconciler {
	t.Helper()
	// we need to set up caches for our reconciler
	runtimeCluster, err := runtimecluster.New(restCfg, func(options *runtimecluster.Options) {
		options.Scheme = testScheme
	})
	require.NoError(t, err)
	go func() {
		startErr := runtimeCluster.Start(ctx)
		require.NoError(t, startErr)
	}()

	cacheClient := runtimeCluster.GetClient()
	reconciler := controllers.NewReconciler(controllers.Params{
		Client:   cacheClient,
		Log:      logger,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(20),
		Config:   cfg,
	})
	err = reconciler.SetupCaches(runtimeCluster)
	require.NoError(t, err)
	return reconciler
}

func sanitizeResourceName(name string) string {
	sanitized := strings.ToLower(name)
	re := regexp.MustCompile("[^a-z0-9-]")
	sanitized = re.ReplaceAllString(sanitized, "-")
	return sanitized
}
