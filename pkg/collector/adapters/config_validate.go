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
	ErrNoPipeline = errors.New("no pipeline available as part of the configuration")
)

func ConfigValidate(logger logr.Logger, config map[interface{}]interface{}) (map[string]bool, error) {
	cfgReceivers, ok := config["receivers"]
	if !ok {
		return nil, ErrNoReceivers
	}
	receivers, ok := cfgReceivers.(map[interface{}]interface{})
	if !ok {
		return nil, ErrReceiversNotAMap
	}
	availableReceivers := map[string]bool{}

	for recvID, recvCfg := range receivers {
		availableReceivers[recvID.(string)] = false
		receiver, ok := recvCfg.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("receiver %q has invalid configuration: %q", recvID, receiver)
		}
	}

	cfgService, ok := config["service"].(map[interface{}]interface{})
	if !ok {
		return nil, ErrNoService
	}

	pipeline, ok := cfgService["pipelines"].(map[interface{}]interface{})
	if !ok {
		return nil, ErrNoPipeline
	}
	availablePipelines := map[string]bool{}

	for pipID := range pipeline {
		availablePipelines[pipID.(string)] = true
	}

	if len(pipeline) > 0 {
		for pipelineID, pipelineCfg := range pipeline {
			//Condition will get information if there are multiple configured pipelines.
			if len(pipelineID.(string)) > 0 {
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
						for _, recKey := range receiversList {
							availableReceivers[recKey.(string)] = true
						}
					}
				}
			}
		}
	}
	return availableReceivers, nil
}
