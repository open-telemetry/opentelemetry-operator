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

package collector

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func Build(params reconcileutil.Params) ([]client.Object, error) {
	var manifests []client.Object
	switch params.Instance.Spec.Mode {
	case v1alpha1.ModeDeployment:
		manifests = append(manifests, Deployment(params.Config, params.Log, params.Instance))
	case v1alpha1.ModeStatefulSet:
		manifests = append(manifests, StatefulSet(params.Config, params.Log, params.Instance))
	case v1alpha1.ModeDaemonSet:
		manifests = append(manifests, DaemonSet(params.Config, params.Log, params.Instance))
	case v1alpha1.ModeSidecar:
		params.Log.V(5).Info("not building sidecar...")
	}
	objects := []reconcileutil.ObjectCreator{
		ConfigMap,
		HorizontalPodAutoscaler,
		reconcileutil.Conformer(ServiceAccount),
		reconcileutil.Conformer(Service),
		reconcileutil.Conformer(HeadlessService),
		reconcileutil.Conformer(MonitoringService),
		Ingress,
	}
	if params.Instance.Spec.Observability.Metrics.EnableMetrics && featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		objects = append(objects, reconcileutil.Conformer(ServiceMonitor))
	}
	for _, object := range objects {
		res, err := object(params.Config, params.Log, params.Instance)
		if err != nil {
			return nil, err
		} else if res != nil && res.DeepCopyObject() != nil {
			manifests = append(manifests, res)
		}
	}
	routes, err := Routes(params.Config, params.Log, params.Instance)
	if err != nil {
		return nil, err
	}
	// NOTE: we cannot just unpack the slice, the type checker doesn't coerce the type correctly.
	for _, route := range routes {
		manifests = append(manifests, route)
	}
	return manifests, nil
}
