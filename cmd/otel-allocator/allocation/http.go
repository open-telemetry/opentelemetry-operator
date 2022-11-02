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

package allocation

import (
	"fmt"
	"net/url"

	"github.com/prometheus/common/model"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

type collectorJSON struct {
	Link string            `json:"_link"`
	Jobs []targetGroupJSON `json:"targets"`
}

type targetGroupJSON struct {
	Targets []string       `json:"targets"`
	Labels  model.LabelSet `json:"labels"`
}

func GetAllTargetsByJob(job string, cMap map[string][]target.Item, allocator Allocator) map[string]collectorJSON {
	displayData := make(map[string]collectorJSON)
	for _, j := range allocator.TargetItems() {
		if j.JobName == job {
			var targetList []target.Item
			targetList = append(targetList, cMap[j.CollectorName+j.JobName]...)

			var targetGroupList []targetGroupJSON

			for _, t := range targetList {
				targetGroupList = append(targetGroupList, targetGroupJSON{
					Targets: []string{t.TargetURL},
					Labels:  t.Label,
				})
			}

			displayData[j.CollectorName] = collectorJSON{Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", url.QueryEscape(j.JobName), j.CollectorName), Jobs: targetGroupList}

		}
	}
	return displayData
}

func GetAllTargetsByCollectorAndJob(collector string, job string, cMap map[string][]target.Item, allocator Allocator) []targetGroupJSON {
	var tgs []targetGroupJSON
	group := make(map[string]target.Item)
	labelSet := make(map[string]model.LabelSet)
	if _, ok := allocator.Collectors()[collector]; ok {
		for _, targetItemArr := range cMap {
			for _, targetItem := range targetItemArr {
				if targetItem.CollectorName == collector && targetItem.JobName == job {
					group[targetItem.Label.String()] = targetItem
					labelSet[targetItem.Hash()] = targetItem.Label
				}
			}
		}
	}
	for _, v := range group {
		tgs = append(tgs, targetGroupJSON{Targets: []string{v.TargetURL}, Labels: labelSet[v.Hash()]})
	}

	return tgs
}
