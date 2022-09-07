package allocation

import (
	"fmt"
	"net/url"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation/strategy"

	"github.com/prometheus/common/model"
)

func GetAllTargetsByJob(job string, cMap map[string][]strategy.TargetItem, allocator strategy.Allocator) map[string]strategy.CollectorJSON {
	displayData := make(map[string]strategy.CollectorJSON)
	for _, j := range allocator.TargetItems() {
		if j.JobName == job {
			var targetList []strategy.TargetItem
			targetList = append(targetList, cMap[j.CollectorName+j.JobName]...)

			var targetGroupList []strategy.TargetGroupJSON

			for _, t := range targetList {
				targetGroupList = append(targetGroupList, strategy.TargetGroupJSON{
					Targets: []string{t.TargetURL},
					Labels:  t.Label,
				})
			}

			displayData[j.CollectorName] = strategy.CollectorJSON{Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", url.QueryEscape(j.JobName), j.CollectorName), Jobs: targetGroupList}

		}
	}
	return displayData
}

func GetAllTargetsByCollectorAndJob(collector string, job string, cMap map[string][]strategy.TargetItem, allocator strategy.Allocator) []strategy.TargetGroupJSON {
	var tgs []strategy.TargetGroupJSON
	group := make(map[string]string)
	labelSet := make(map[string]model.LabelSet)
	for colName, _ := range allocator.Collectors() {
		if colName == collector {
			for _, targetItemArr := range cMap {
				for _, targetItem := range targetItemArr {
					if targetItem.CollectorName == collector && targetItem.JobName == job {
						group[targetItem.Label.String()] = targetItem.TargetURL
						labelSet[targetItem.TargetURL] = targetItem.Label
					}
				}
			}
		}
	}

	for _, v := range group {
		tgs = append(tgs, strategy.TargetGroupJSON{Targets: []string{v}, Labels: labelSet[v]})
	}

	return tgs
}
