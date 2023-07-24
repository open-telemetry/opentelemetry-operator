package collector

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"
)

func Build(params reconcileutil.Params) ([]client.Object, error) {
	var manifests []client.Object
	switch params.Instance.Spec.Mode {
	case "deployment":
		manifests = append(manifests, Deployment(params.Config, params.Log, params.Instance))
	case "statefulset":
		manifests = append(manifests, StatefulSet(params.Config, params.Log, params.Instance))
	case "daemonset":
		manifests = append(manifests, DaemonSet(params.Config, params.Log, params.Instance))
	}
	objects := []reconcileutil.ObjectCreator{
		HorizontalPodAutoscaler,
		ServiceAccount,
		Service,
		HeadlessService,
		MonitoringService,
		Ingress,
	}
	for _, object := range objects {
		manifests = append(manifests, object(params.Config, params.Log, params.Instance))
	}
	routes := Routes(params.Config, params.Log, params.Instance)
	// NOTE: we cannot just unpack the slice, the type checker doesn't coerce the type correctly.
	for _, route := range routes {
		manifests = append(manifests, route)
	}
	return nil, nil
}
