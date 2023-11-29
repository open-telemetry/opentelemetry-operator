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

import "fmt"

// Following Otel Doc: Configuring a receiver does not enable it. The receivers are enabled via pipelines within the service section.
// getEnabledComponents returns all enabled components as a true flag set. If it can't find any receiver, it will return a nil interface.
func getEnabledComponents(config map[interface{}]interface{}, componentType ComponentType) map[interface{}]bool {
	componentTypePlural := fmt.Sprintf("%ss", componentType)
	cfgComponents, ok := config[componentTypePlural]
	if !ok {
		return nil
	}
	components, ok := cfgComponents.(map[interface{}]interface{})
	if !ok {
		return nil
	}
	availableComponents := map[interface{}]bool{}

	for compID := range components {

		//Safe Cast
		componentID, withComponent := compID.(string)
		if !withComponent {
			return nil
		}
		//Getting all components present in the components (exporters,receivers...) section and setting them to false.
		availableComponents[componentID] = false
	}

	cfgService, withService := config["service"].(map[interface{}]interface{})
	if !withService {
		return nil
	}

	pipeline, withPipeline := cfgService["pipelines"].(map[interface{}]interface{})
	if !withPipeline {
		return nil
	}
	availablePipelines := map[string]bool{}

	for pipID := range pipeline {
		//Safe Cast
		pipelineID, existsPipeline := pipID.(string)
		if !existsPipeline {
			return nil
		}
		//Getting all the available pipelines.
		availablePipelines[pipelineID] = true
	}

	if len(pipeline) > 0 {
		for pipelineID, pipelineCfg := range pipeline {
			//Safe Cast
			pipelineV, withPipelineCfg := pipelineID.(string)
			if !withPipelineCfg {
				continue
			}
			//Condition will get information if there are multiple configured pipelines.
			if len(pipelineV) > 0 {
				pipelineDesc, ok := pipelineCfg.(map[interface{}]interface{})
				if !ok {
					return nil
				}
				for pipSpecID, pipSpecCfg := range pipelineDesc {
					if pipSpecID.(string) == componentTypePlural {
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
	}
	return availableComponents
}
