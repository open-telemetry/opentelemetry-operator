package targetallocator

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"
)

func Build(params reconcileutil.Params) ([]client.Object, error) {
	var manifests []client.Object
	if !params.Instance.Spec.TargetAllocator.Enabled {
		return nil, nil
	}
	objects := []reconcileutil.ObjectCreator{
		Deployment,
		ServiceAccount,
		Service,
	}
	for _, object := range objects {
		manifests = append(manifests, object(params.Config, params.Log, params.Instance))
	}
	return manifests, nil
}
