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
	"errors"
	"fmt"

	"github.com/go-logr/logr"
)

var (
	errNoPipeline = errors.New("no pipeline available as part of the configuration")
)

//Following Otel Doc: Configuring a receiver does not enable it. The receivers are enabled via pipelines within the service section.
//ConfigValidate returns all receivers, setting them as true for enabled and false for non-configured services in pipeline set.
func GetEnabledReceivers(logger logr.Logger, config map[interface{}]interface{}) (map[interface{}]bool, error) {
	cfgReceivers, ok := config["receivers"]
	if !ok {
		return nil, ErrNoReceivers
	}
	receivers, ok := cfgReceivers.(map[interface{}]interface{})
	if !ok {
		return nil, ErrReceiversNotAMap
	}
	availableReceivers := map[interface{}]bool{}

	for recvID := range receivers {

		//Safe Cast
		receiverID, ok := recvID.(string)
		if !ok {
			return nil, fmt.Errorf("ReceiverID is not a string: %v", receiverID)
		}
		//Getting all receivers present in the receivers section and setting them to false.
		availableReceivers[receiverID] = false
	}

	cfgService, ok := config["service"].(map[interface{}]interface{})
	if !ok {
		return nil, errNoService
	}

	pipeline, ok := cfgService["pipelines"].(map[interface{}]interface{})
	if !ok {
		return nil, errNoPipeline
	}
	availablePipelines := map[string]bool{}

	for pipID := range pipeline {
		//Safe Cast
		pipelineID, ok := pipID.(string)
		if !ok {
			return nil, fmt.Errorf("PipelineID is not a string: %v", pipelineID)
		}
		//Getting all the available pipelines.
		availablePipelines[pipelineID] = true
	}

	if len(pipeline) > 0 {
		for pipelineID, pipelineCfg := range pipeline {
			//Safe Cast
			pipelineV, ok := pipelineID.(string)
			if !ok {
				return nil, fmt.Errorf("PipelineID is not a string: %v", pipelineV)
			}
			//Condition will get information if there are multiple configured pipelines.
			if len(pipelineV) > 0 {
				pipelineDesc, ok := pipelineCfg.(map[interface{}]interface{})
				if !ok {
					return nil, fmt.Errorf("pipeline was not properly configured")
				}
				for pipSpecID, pipSpecCfg := range pipelineDesc {
					if pipSpecID.(string) == "receivers" {
						receiversList, ok := pipSpecCfg.([]interface{})
						if !ok {
							return nil, fmt.Errorf("no receivers on pipeline configuration %q", receiversList...)
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
									return nil, fmt.Errorf("ReceiverKey is not a string: %v", receiverKey)
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
	return availableReceivers, nil
}
