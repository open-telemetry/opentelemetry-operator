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

package operatormetrics

import (
	"context"
	"fmt"
	"os"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	// namespaceFile is the path to the namespace file for the service account.
	namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	// caBundleConfigMap declares the name of the config map for the CA bundle.
	caBundleConfigMap = "serving-certs-ca-bundle"

	// prometheusCAFile declares the path for prometheus CA file for service monitors in OpenShift.
	prometheusCAFile = fmt.Sprintf("/etc/prometheus/configmaps/%s/service-ca.crt", caBundleConfigMap)

	// nolint #nosec
	// bearerTokenFile declares the path for bearer token file for service monitors.
	bearerTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// openshiftInClusterMonitoringNamespace declares the namespace for the OpenShift in-cluster monitoring.
	openshiftInClusterMonitoringNamespace = "openshift-monitoring"
)

var _ manager.Runnable = &OperatorMetrics{}

type OperatorMetrics struct {
	kubeClient client.Client
}

func NewOperatorMetrics(config *rest.Config, scheme *runtime.Scheme) (OperatorMetrics, error) {
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return OperatorMetrics{}, err
	}

	return OperatorMetrics{
		kubeClient: kubeClient,
	}, nil
}

func (om OperatorMetrics) Start(ctx context.Context) error {
	rawNamespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		return fmt.Errorf("error reading namespace file: %w", err)
	}
	namespace := string(rawNamespace)

	var tlsConfig *monitoringv1.TLSConfig

	if om.caConfigMapExists() {
		serviceName := fmt.Sprintf("opentelemetry-operator-controller-manager-metrics-service.%s.svc", namespace)

		tlsConfig = &monitoringv1.TLSConfig{
			CAFile: prometheusCAFile,
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				ServerName: &serviceName,
			},
		}
	} else {
		t := true
		tlsConfig = &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				// kube-rbac-proxy uses a self-signed cert by default
				InsecureSkipVerify: &t,
			},
		}
	}

	sm := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opentelemetry-operator-metrics-monitor",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "opentelemetry-operator",
				"app.kubernetes.io/part-of": "opentelemetry-operator",
				"control-plane":             "controller-manager",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "opentelemetry-operator",
				},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					BearerTokenFile: bearerTokenFile,
					Interval:        "30s",
					Path:            "/metrics",
					Scheme:          "https",
					ScrapeTimeout:   "10s",
					TargetPort:      &intstr.IntOrString{IntVal: 8443},
					TLSConfig:       tlsConfig,
				},
			},
		},
	}

	err = om.kubeClient.Create(ctx, &sm)
	if err != nil {
		return fmt.Errorf("error creating service monitor: %w", err)
	}

	<-ctx.Done()

	return om.kubeClient.Delete(ctx, &sm)
}

func (om OperatorMetrics) NeedLeaderElection() bool {
	return true
}

func (om OperatorMetrics) caConfigMapExists() bool {
	return om.kubeClient.Get(context.Background(), client.ObjectKey{
		Name:      caBundleConfigMap,
		Namespace: openshiftInClusterMonitoringNamespace,
	}, &corev1.ConfigMap{},
	) == nil
}
