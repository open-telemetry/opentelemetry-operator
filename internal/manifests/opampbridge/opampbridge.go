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

package opampbridge

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

const (
	ComponentOpAMPBridge = "opentelemetry-opamp-bridge"
)

// Build creates the manifest for the OpAMPBridge resource.
func Build(params manifests.Params) ([]client.Object, error) {
	var resourceManifests []client.Object
	resourceFactories := []manifests.K8sManifestFactory{
		manifests.FactoryWithoutError(Deployment),
		manifests.Factory(ConfigMap),
		manifests.FactoryWithoutError(ServiceAccount),
		manifests.FactoryWithoutError(Service),
	}
	for _, factory := range resourceFactories {
		res, err := factory(params)
		if err != nil {
			return nil, err
		} else if manifests.ObjectIsNotNil(res) {
			resourceManifests = append(resourceManifests, res)
		}
	}
	return resourceManifests, nil
}
