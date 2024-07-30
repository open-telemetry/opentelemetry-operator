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

package controllers_test

import (
	"context"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyV1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
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
				params:  testCollectorWithPDB(1, 0),
				updates: []v1alpha1.OpenTelemetryCollector{testCollectorWithPDB(0, 1)},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := policyV1.PodDisruptionBudget{}
							exists, pdbErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.Name), params.Namespace))
							assert.NoError(t, pdbErr)
							assert.Equal(t, int32(1), actual.Spec.MinAvailable.IntVal)
							assert.Nil(t, actual.Spec.MaxUnavailable)
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
							assert.Nil(t, actual.Spec.MinAvailable)
							assert.Equal(t, int32(1), actual.Spec.MaxUnavailable.IntVal)
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
							exists, err = populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.False(t, exists)
							exists, err = populateObjectIfExists(t, &appsv1.Deployment{}, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
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
							actual := v1.ConfigMap{}
							exists, err = populateObjectIfExists(t, &appsv1.Deployment{}, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, namespacedObjectName(naming.TargetAllocatorServiceAccount(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							promConfig, err := ta.ConfigToPromConfig(testCollectorAssertNoErr(t, "test-stateful-ta", baseTaImage, promFile).Spec.Config)
							assert.NoError(t, err)

							taConfig := make(map[interface{}]interface{})
							taConfig["collector_selector"] = metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/instance":   "default.test-stateful-ta",
									"app.kubernetes.io/managed-by": "opentelemetry-operator",
									"app.kubernetes.io/component":  "opentelemetry-collector",
									"app.kubernetes.io/part-of":    "opentelemetry",
								},
							}
							taConfig["config"] = promConfig["config"]
							taConfig["allocation_strategy"] = "consistent-hashing"
							taConfig["filter_strategy"] = "relabel-config"
							taConfigYAML, _ := yaml.Marshal(taConfig)
							assert.Equal(t, string(taConfigYAML), actual.Data["targetallocator.yaml"])
							assert.NotContains(t, actual.Data["targetallocator.yaml"], "0.0.0.0:10100")
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
							actual := v1.ConfigMap{}
							exists, err = populateObjectIfExists(t, &appsv1.Deployment{}, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Contains(t, actual.Data["targetallocator.yaml"], "0.0.0.0:10100")
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check[v1alpha1.OpenTelemetryCollector]{
						func(t *testing.T, params v1alpha1.OpenTelemetryCollector) {
							actual := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &actual, namespacedObjectName(naming.TargetAllocator(params.Name), params.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, actual.Spec.Template.Spec.Containers[0].Image, updatedTaImage)
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
			reconciler := controllers.NewReconciler(controllers.Params{
				Client:   k8sClient,
				Log:      logger,
				Scheme:   testScheme,
				Recorder: record.NewFakeRecorder(20),
				Config: config.New(
					config.WithCollectorImage("default-collector"),
					config.WithTargetAllocatorImage("default-ta-allocator"),
					config.WithOpenShiftRoutesAvailability(openshift.RoutesAvailable),
					config.WithPrometheusCRAvailability(prometheus.Available),
				),
			})

			assert.True(t, len(tt.want) > 0, "must have at least one group of checks to run")
			firstCheck := tt.want[0]
			// Check for this before create, otherwise it's blown away.
			deletionTimestamp := tt.args.params.GetDeletionTimestamp()
			createErr := k8sClient.Create(testContext, &tt.args.params)
			if !firstCheck.validateErr(t, createErr) {
				return
			}
			if deletionTimestamp != nil {
				err := k8sClient.Delete(testContext, &tt.args.params, client.PropagationPolicy(metav1.DeletePropagationForeground))
				assert.NoError(t, err)
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

	reconciler := controllers.NewReconciler(controllers.Params{
		Client:   k8sClient,
		Log:      logger,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(20),
		Config: config.New(
			config.WithCollectorImage("default-collector"),
			config.WithTargetAllocatorImage("default-ta-allocator"),
			config.WithRBACPermissions(autoRBAC.Available),
		),
	})

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
