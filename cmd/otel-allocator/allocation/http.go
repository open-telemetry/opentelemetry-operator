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

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

type collectorJSON struct {
	Link string         `json:"_link"`
	Jobs []*target.Item `json:"targets"`
}

// GetAllTargetsByJob is a relatively expensive call that is usually only used for debugging purposes.
func GetAllTargetsByJob(allocator Allocator, job string) map[string]collectorJSON {
	displayData := make(map[string]collectorJSON)
	for _, col := range allocator.Collectors() {
		items := allocator.GetTargetsForCollectorAndJob(col.Name, job)
		displayData[col.Name] = collectorJSON{Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", url.QueryEscape(job), col.Name), Jobs: items}
	}
	return displayData
}

func GetAllTargetsByCollectorAndJob(allocator Allocator, collector string, job string) []*target.Item {
	return allocator.GetTargetsForCollectorAndJob(collector, job)
}
