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

package adapters

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
)

// Following Otel Doc: Configuring a receiver does not enable it. The receivers are enabled via pipelines within the service section.
// getEnabledComponents returns all enabled components as a true flag set. If it can't find any receiver, it will return a nil interface.
func getEnabledComponents(config v1alpha2.Service, componentType ComponentType) map[string]bool {
	availableComponents := map[string]bool{}
	componentTypePlural := fmt.Sprintf("%ss", componentType.String())
	for pipelineID, pipelineCfg := range config.Pipelines.Object {
		//Condition will get information if there are multiple configured pipelines.
		if len(pipelineID) > 0 {
			pipelineDesc, ok := pipelineCfg.(map[string]interface{})
			if !ok {
				return nil
			}
			for pipSpecID, pipSpecCfg := range pipelineDesc {
				if pipSpecID == componentTypePlural {
					receiversList, ok := pipSpecCfg.([]interface{})
					if !ok {
						continue
					}
					// If receiversList is empty means that we haven't any enabled Receiver.
					if len(receiversList) == 0 {
						availableComponents = nil
					} else {
						// All enabled receivers will be set as true
						for _, comKey := range receiversList {
							//Safe Cast
							receiverKey, ok := comKey.(string)
							if !ok {
								return nil
							}
							availableComponents[receiverKey] = true
						}
					}
					//Removing all non-enabled receivers
					for comID, comKey := range availableComponents {
						if !(comKey) {
							delete(availableComponents, comID)
						}
					}
				}
			}
		}
	}
	return availableComponents
}
