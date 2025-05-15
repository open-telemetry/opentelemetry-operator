// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	_ "embed"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// The dashboard is created manually following the syntax from Grafana 5. For development purposes, this dashboard can be created just by loading the JSON file
// in a ConfigMap from the openshift-config-managed and adding the console.openshift.io/dashboard=true label.
//
//go:embed metrics-dashboard.json
var dashboardContent string

const (
	openshiftDashboardsNamespace = "openshift-config-managed"
	configMapName                = "opentelemetry-collector"
)

type dashboardManagement struct {
	clientset kubernetes.Interface
}

var _ manager.Runnable = (*dashboardManagement)(nil)

func NewDashboardManagement(clientset kubernetes.Interface) manager.Runnable {
	return dashboardManagement{
		clientset: clientset,
	}
}

func (d dashboardManagement) Start(ctx context.Context) error {
	cm := corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      configMapName,
			Namespace: openshiftDashboardsNamespace,
			Labels: map[string]string{
				"console.openshift.io/dashboard": "true",
			},
		},
		Data: map[string]string{
			"otel.json": dashboardContent,
		},
	}

	_, err := d.clientset.CoreV1().ConfigMaps(openshiftDashboardsNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err := d.clientset.CoreV1().ConfigMaps(openshiftDashboardsNamespace).Create(ctx, &cm, metav1.CreateOptions{})
			if err != nil {
				return nil
			}
		}
	} else {
		// config map already exists, update it
		_, err := d.clientset.CoreV1().ConfigMaps(openshiftDashboardsNamespace).Update(ctx, &cm, metav1.UpdateOptions{})
		if err != nil {
			return nil
		}
	}

	<-ctx.Done()

	return d.clientset.CoreV1().ConfigMaps(openshiftDashboardsNamespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
}

func (d dashboardManagement) NeedLeaderElection() bool {
	return true
}
