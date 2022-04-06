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
	"github.com/go-logr/logr"
)

//Following Otel Doc: Configuring a receiver does not enable it. The receivers are enabled via pipelines within the service section.
//GetEnabledReceivers returns all enabled receivers as a true flag set. If it can't find any receiver, it will return a nil interface.
func GetEnabledReceivers(logger logr.Logger, config map[interface{}]interface{}) map[interface{}]bool {
	cfgReceivers, ok := config["receivers"]
	if !ok {
		return nil
	}
	receivers, ok := cfgReceivers.(map[interface{}]interface{})
	if !ok {
		return nil
	}
	availableReceivers := map[interface{}]bool{}

	for recvID := range receivers {

		//Safe Cast
		receiverID, ok := recvID.(string)
		if !ok {
			return nil
		}
		//Getting all receivers present in the receivers section and setting them to false.
		availableReceivers[receiverID] = false
	}

	cfgService, ok := config["service"].(map[interface{}]interface{})
	if !ok {
		return nil
	}

	pipeline, ok := cfgService["pipelines"].(map[interface{}]interface{})
	if !ok {
		return nil
	}
	availablePipelines := map[string]bool{}

	for pipID := range pipeline {
		//Safe Cast
		pipelineID, ok := pipID.(string)
		if !ok {
			return nil
		}
		//Getting all the available pipelines.
		availablePipelines[pipelineID] = true
	}

	if len(pipeline) > 0 {
		for pipelineID, pipelineCfg := range pipeline {
			//Safe Cast
			pipelineV, ok := pipelineID.(string)
			if !ok {
				continue
			}
			//Condition will get information if there are multiple configured pipelines.
			if len(pipelineV) > 0 {
				pipelineDesc, ok := pipelineCfg.(map[interface{}]interface{})
				if !ok {
					return nil
				}
				for pipSpecID, pipSpecCfg := range pipelineDesc {
					if pipSpecID.(string) == "receivers" {
						receiversList, ok := pipSpecCfg.([]interface{})
						if !ok {
							continue
						}
						// If receiversList is empty means that we haven't any enabled Receiver.
						if len(receiversList) == 0 {
							availableReceivers = nil
						} else {
							// All enabled receivers will be set as true
							for _, recKey := range receiversList {
								//Safe Cast
								receiverKey, ok := recKey.(string)
								if !ok {
									return nil
								}
								availableReceivers[receiverKey] = true
							}
						}
						//Removing all non-enabled receivers
						for recID, recKey := range availableReceivers {
							if !(recKey) {
								delete(availableReceivers, recID)
							}
						}
					}
				}
			}
		}
	}
	return availableReceivers
}
