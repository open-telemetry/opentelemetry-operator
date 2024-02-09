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

package target

import (
	"fmt"
	"net/url"

	"github.com/prometheus/common/model"
)

// nodeLabels are labels that are used to identify the node on which the given
// target is residing. To learn more about these labels, please refer to:
// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config
var nodeLabels = []model.LabelName{
	"__meta_kubernetes_pod_node_name",
	"__meta_kubernetes_node_name",
	"__meta_kubernetes_endpoint_node_name",
}

// LinkJSON This package contains common structs and methods that relate to scrape targets.
type LinkJSON struct {
	Link string `json:"_link"`
}

type Item struct {
	JobName       string         `json:"-"`
	Link          LinkJSON       `json:"-"`
	TargetURL     []string       `json:"targets"`
	Labels        model.LabelSet `json:"labels"`
	CollectorName string         `json:"-"`
	hash          string
}

func (t *Item) Hash() string {
	return t.hash
}

func (t *Item) GetNodeName() string {
	for _, label := range nodeLabels {
		if val, ok := t.Labels[label]; ok {
			return string(val)
		}
	}

	return ""
}

// NewItem Creates a new target item.
// INVARIANTS:
// * Item fields must not be modified after creation.
// * Item should only be made via its constructor, never directly.
func NewItem(jobName string, targetURL string, label model.LabelSet, collectorName string) *Item {
	return &Item{
		JobName:       jobName,
		Link:          LinkJSON{Link: fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(jobName))},
		hash:          jobName + targetURL + label.Fingerprint().String(),
		TargetURL:     []string{targetURL},
		Labels:        label,
		CollectorName: collectorName,
	}
}
