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
		ConfigMap,
		Deployment,
		reconcileutil.Conformer(ServiceAccount),
		Service,
	}
	for _, object := range objects {
		res, err := object(params.Config, params.Log, params.Instance)
		if err != nil {
			return nil, err
		} else if res != nil && res.DeepCopyObject() != nil {
			manifests = append(manifests, res)
		}
	}
	return manifests, nil
}
