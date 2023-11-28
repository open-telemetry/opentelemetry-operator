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
	extraPorts = v1.ServicePort{
		Name:       "port-web",
		Protocol:   "TCP",
		Port:       8080,
		TargetPort: intstr.FromInt32(8080),
	}
)

type check func(t *testing.T, params manifests.Params)

func newParamsAssertNoErr(t *testing.T, taContainerImage string, file string) manifests.Params {
	p, err := newParams(taContainerImage, file)
	assert.NoError(t, err)
	if len(taContainerImage) == 0 {
		p.OtelCol.Spec.TargetAllocator.Enabled = false
	}
	return p
}

func TestOpenTelemetryCollectorReconciler_Reconcile(t *testing.T) {
	addedMetadataDeployment := paramsWithMode(v1alpha1.ModeDeployment)
	addedMetadataDeployment.OtelCol.Labels = map[string]string{
		labelName: labelVal,
	}
	addedMetadataDeployment.OtelCol.Annotations = map[string]string{
		annotationName: annotationVal,
	}
	deploymentExtraPorts := paramsWithModeAndReplicas(v1alpha1.ModeDeployment, 3)
	deploymentExtraPorts.OtelCol.Spec.Ports = append(deploymentExtraPorts.OtelCol.Spec.Ports, extraPorts)
	ingressParams := newParamsAssertNoErr(t, "", testFileIngress)
	ingressParams.OtelCol.Spec.Ingress.Type = "ingress"
	updatedIngressParams := newParamsAssertNoErr(t, "", testFileIngress)
	updatedIngressParams.OtelCol.Spec.Ingress.Type = "ingress"
	updatedIngressParams.OtelCol.Spec.Ingress.Annotations = map[string]string{"blub": "blob"}
	updatedIngressParams.OtelCol.Spec.Ingress.Hostname = expectHostname
	routeParams := newParamsAssertNoErr(t, "", testFileIngress)
	routeParams.OtelCol.Spec.Ingress.Type = v1alpha1.IngressTypeRoute
	routeParams.OtelCol.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeInsecure
	updatedRouteParams := newParamsAssertNoErr(t, "", testFileIngress)
	updatedRouteParams.OtelCol.Spec.Ingress.Type = v1alpha1.IngressTypeRoute
	updatedRouteParams.OtelCol.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeInsecure
	updatedRouteParams.OtelCol.Spec.Ingress.Hostname = expectHostname
	deletedParams := paramsWithMode(v1alpha1.ModeDeployment)
	now := metav1.NewTime(time.Now())
	deletedParams.OtelCol.DeletionTimestamp = &now

	type args struct {
		params manifests.Params
		// an optional list of updates to supply after the initial object
		updates []manifests.Params
	}
	type want struct {
		// result check
		result controllerruntime.Result
		// a check to run against the current state applied
		checks []check
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
				updates: []manifests.Params{deploymentExtraPorts},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(2), *d.Spec.Replicas)
							assert.Contains(t, d.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							exists, err = populateObjectIfExists(t, &v1.Service{}, namespacedObjectName(naming.Service(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, namespacedObjectName(naming.ServiceAccount(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(3), *d.Spec.Replicas)
							// confirm that we don't remove annotations and labels even if we don't set them
							assert.Contains(t, d.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							actual := v1.Service{}
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.Service(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Contains(t, actual.Spec.Ports, extraPorts)
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
				params:  paramsWithMode("bad"),
				updates: []manifests.Params{},
			},
			want: []want{
				{
					result:  controllerruntime.Result{},
					checks:  []check{},
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
				params:  newParamsAssertNoErr(t, baseTaImage, testFileIngress),
				updates: []manifests.Params{},
			},
			want: []want{
				{
					result:  controllerruntime.Result{},
					checks:  []check{},
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
				updates: []manifests.Params{updatedIngressParams},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							d := networkingv1.Ingress{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Ingress(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							d := networkingv1.Ingress{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.Ingress(params.OtelCol.Name), params.OtelCol.Namespace))
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
				updates: []manifests.Params{updatedRouteParams},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							got := routev1.Route{}
							nsn := types.NamespacedName{Namespace: params.OtelCol.Namespace, Name: "otlp-grpc-test-route"}
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
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							got := routev1.Route{}
							nsn := types.NamespacedName{Namespace: params.OtelCol.Namespace, Name: "otlp-grpc-test-route"}
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
				params:  paramsWithHPA(3, 5),
				updates: []manifests.Params{paramsWithHPA(1, 9)},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							actual := autoscalingv2.HorizontalPodAutoscaler{}
							exists, hpaErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.OtelCol.Name), params.OtelCol.Namespace))
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
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							actual := autoscalingv2.HorizontalPodAutoscaler{}
							exists, hpaErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.OtelCol.Name), params.OtelCol.Namespace))
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
				params:  paramsWithPolicy(1, 0),
				updates: []manifests.Params{paramsWithPolicy(0, 1)},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							actual := policyV1.PodDisruptionBudget{}
							exists, pdbErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.OtelCol.Name), params.OtelCol.Namespace))
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
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							actual := policyV1.PodDisruptionBudget{}
							exists, pdbErr := populateObjectIfExists(t, &actual, namespacedObjectName(naming.HorizontalPodAutoscaler(params.OtelCol.Name), params.OtelCol.Namespace))
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
				params: paramsWithMode(v1alpha1.ModeDaemonSet),
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							exists, err := populateObjectIfExists(t, &appsv1.DaemonSet{}, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
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
				params: paramsWithMode(v1alpha1.ModeStatefulSet),
				updates: []manifests.Params{
					newParamsAssertNoErr(t, baseTaImage, promFile),
					newParamsAssertNoErr(t, baseTaImage, updatedPromFile),
					newParamsAssertNoErr(t, updatedTaImage, updatedPromFile),
				},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &appsv1.StatefulSet{}, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							// Check the TA doesn't exist
							exists, err = populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.False(t, exists)
							exists, err = populateObjectIfExists(t, &appsv1.Deployment{}, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.False(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							actual := v1.ConfigMap{}
							exists, err = populateObjectIfExists(t, &appsv1.Deployment{}, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, namespacedObjectName(naming.TargetAllocatorServiceAccount(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)

							promConfig, err := ta.ConfigToPromConfig(newParamsAssertNoErr(t, baseTaImage, promFile).OtelCol.Spec.Config)
							assert.NoError(t, err)

							taConfig := make(map[interface{}]interface{})
							taConfig["label_selector"] = map[string]string{
								"app.kubernetes.io/instance":   "default.test",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/part-of":    "opentelemetry",
							}
							taConfig["config"] = promConfig["config"]
							taConfig["allocation_strategy"] = "least-weighted"
							taConfig["prometheus_cr"] = map[string]string{
								"scrape_interval": "30s",
							}
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
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							exists, err := populateObjectIfExists(t, &v1.ConfigMap{}, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							actual := v1.ConfigMap{}
							exists, err = populateObjectIfExists(t, &appsv1.Deployment{}, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
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
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							actual := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &actual, namespacedObjectName(naming.TargetAllocator(params.OtelCol.Name), params.OtelCol.Namespace))
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
				updates: []manifests.Params{},
			},
			want: []want{
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							o := v1alpha1.OpenTelemetryCollector{}
							exists, err := populateObjectIfExists(t, &o, namespacedObjectName(naming.Collector(params.OtelCol.Name), params.OtelCol.Namespace))
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
			nsn := types.NamespacedName{Name: tt.args.params.OtelCol.Name, Namespace: tt.args.params.OtelCol.Namespace}
			reconciler := controllers.NewReconciler(controllers.Params{
				Client:   k8sClient,
				Log:      logger,
				Scheme:   testScheme,
				Recorder: record.NewFakeRecorder(20),
				Config: config.New(
					config.WithCollectorImage("default-collector"),
					config.WithTargetAllocatorImage("default-ta-allocator"),
					config.WithOpenShiftRoutesAvailability(openshift.RoutesAvailable),
				),
			})

			assert.True(t, len(tt.want) > 0, "must have at least one group of checks to run")
			firstCheck := tt.want[0]
			// Check for this before create, otherwise it's blown away.
			deletionTimestamp := tt.args.params.OtelCol.GetDeletionTimestamp()
			createErr := k8sClient.Create(testContext, &tt.args.params.OtelCol)
			if !firstCheck.validateErr(t, createErr) {
				return
			}
			if deletionTimestamp != nil {
				err := k8sClient.Delete(testContext, &tt.args.params.OtelCol, client.PropagationPolicy(metav1.DeletePropagationForeground))
				assert.NoError(t, err)
			}
			req := k8sreconcile.Request{
				NamespacedName: nsn,
			}
			got, reconcileErr := reconciler.Reconcile(testContext, req)
			if !firstCheck.wantErr(t, reconcileErr) {
				require.NoError(t, k8sClient.Delete(testContext, &tt.args.params.OtelCol))
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

				updateParam.OtelCol.SetResourceVersion(existing.ResourceVersion)
				updateParam.OtelCol.SetUID(existing.UID)
				err = k8sClient.Update(testContext, &updateParam.OtelCol)
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
				require.NoError(t, k8sClient.Delete(testContext, &tt.args.params.OtelCol))
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
	deploymentExtraPorts.OpAMPBridge.Spec.Ports = append(deploymentExtraPorts.OpAMPBridge.Spec.Ports, extraPorts)

	type args struct {
		params manifests.Params
		// an optional list of updates to supply after the initial object
		updates []manifests.Params
	}
	type want struct {
		// result check
		result controllerruntime.Result
		// a check to run against the current state applied
		checks []check
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
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.OpAMPBridge(params.OpAMPBridge.Name), params.OpAMPBridge.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Equal(t, int32(1), *d.Spec.Replicas)
							assert.Contains(t, d.Spec.Template.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							exists, err = populateObjectIfExists(t, &v1.Service{}, namespacedObjectName(naming.OpAMPBridgeService(params.OpAMPBridge.Name), params.OpAMPBridge.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							exists, err = populateObjectIfExists(t, &v1.ServiceAccount{}, namespacedObjectName(naming.ServiceAccount(params.OpAMPBridge.Name), params.OpAMPBridge.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
						},
					},
					wantErr:     assert.NoError,
					validateErr: assert.NoError,
				},
				{
					result: controllerruntime.Result{},
					checks: []check{
						func(t *testing.T, params manifests.Params) {
							d := appsv1.Deployment{}
							exists, err := populateObjectIfExists(t, &d, namespacedObjectName(naming.OpAMPBridge(params.OpAMPBridge.Name), params.OpAMPBridge.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							// confirm that we don't remove annotations and labels even if we don't set them
							assert.Contains(t, d.Spec.Template.Annotations, annotationName)
							assert.Contains(t, d.Labels, labelName)
							actual := v1.Service{}
							exists, err = populateObjectIfExists(t, &actual, namespacedObjectName(naming.OpAMPBridgeService(params.OpAMPBridge.Name), params.OpAMPBridge.Namespace))
							assert.NoError(t, err)
							assert.True(t, exists)
							assert.Contains(t, actual.Spec.Ports, extraPorts)
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
				check(t, tt.args.params)
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
					check(t, updateParam)
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

func namespacedObjectName(name string, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}
