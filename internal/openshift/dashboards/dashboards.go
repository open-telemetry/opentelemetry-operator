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

package openshift

import (
	"context"
	_ "embed"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// The dashboard is created manually following the syntax from Grafana 5. For development purposes, this dashboard can be created just loading the JSON file
// in a ConfigMap from the openshift-config-managed and adding the console.openshift.io/dashboard=true label.
//
//go:embed metrics-dashboard.json
var dashboardContent string

const (
	openshiftDashboardsNamespace = "openshift-config-managed"
	configMapName                = "opentelemetry-collector"
)

func CreateOpenShiftDashboard(clientset kubernetes.Interface) error {
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

	_, err := clientset.CoreV1().ConfigMaps(openshiftDashboardsNamespace).Create(context.TODO(), &cm, metav1.CreateOptions{})
	return err
}

func DeleteOpenShiftDashboard(clientset kubernetes.Interface, logger logr.Logger) {
	err := clientset.CoreV1().ConfigMaps(openshiftDashboardsNamespace).Delete(context.TODO(), configMapName, metav1.DeleteOptions{})
	if err != nil {
		logger.Error(err, "it was not possible to remove the dashboards configmap", "name", configMapName, "namespace", openshiftDashboardsNamespace)
	}
}
