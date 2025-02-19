// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatormetrics

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	log        logr.Logger
}

func NewOperatorMetrics(config *rest.Config, scheme *runtime.Scheme, log logr.Logger) (OperatorMetrics, error) {
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return OperatorMetrics{}, err
	}

	return OperatorMetrics{
		kubeClient: kubeClient,
		log:        log,
	}, nil
}

func (om OperatorMetrics) Start(ctx context.Context) error {
	err := om.createOperatorMetricsServiceMonitor(ctx)
	if err != nil {
		om.log.Error(err, "error creating Service Monitor for operator metrics")
	}

	return nil
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

func (om OperatorMetrics) getOwnerReferences(ctx context.Context, namespace string) (metav1.OwnerReference, error) {
	var deploymentList appsv1.DeploymentList

	listOptions := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/name": "opentelemetry-operator",
			"control-plane":          "controller-manager",
		}),
	}

	err := om.kubeClient.List(ctx, &deploymentList, listOptions...)
	if err != nil {
		return metav1.OwnerReference{}, err
	}

	if len(deploymentList.Items) == 0 {
		return metav1.OwnerReference{}, fmt.Errorf("no deployments found with the specified label")
	}
	deployment := &deploymentList.Items[0]

	ownerRef := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       deployment.Name,
		UID:        deployment.UID,
	}

	return ownerRef, nil
}

func (om OperatorMetrics) createOperatorMetricsServiceMonitor(ctx context.Context) error {
	rawNamespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		return fmt.Errorf("error reading namespace file: %w", err)
	}
	namespace := string(rawNamespace)

	ownerRef, err := om.getOwnerReferences(ctx, namespace)
	if err != nil {
		return fmt.Errorf("error getting owner references: %w", err)
	}

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
			OwnerReferences: []metav1.OwnerReference{ownerRef},
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
	// The ServiceMonitor can be already there if this is a restart
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	<-ctx.Done()

	return om.kubeClient.Delete(ctx, &sm)
}
