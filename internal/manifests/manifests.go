package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"
)

func BuildAll(params reconcileutil.Params) ([]client.Object, error) {
	builders := []reconcileutil.Builder{
		collector.Build,
		targetallocator.Build,
	}
	var manifests []client.Object
	for _, builder := range builders {
		objs, err := builder(params)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, objs...)
	}
	return manifests, nil
}
