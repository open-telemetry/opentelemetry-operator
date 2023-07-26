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
		reconcileutil.Conformer(Service),
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
