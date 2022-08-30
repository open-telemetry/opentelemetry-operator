package allocation

import (
	"fmt"
	"net/url"

	"github.com/prometheus/common/model"
)

type LinkJSON struct {
	Link string `json:"_link"`
}

type collectorJSON struct {
	Link string            `json:"_link"`
	Jobs []targetGroupJSON `json:"targets"`
}

type targetGroupJSON struct {
	Targets []string       `json:"targets"`
	Labels  model.LabelSet `json:"labels"`
}

func GetAllTargetsByJob(job string, cMap map[string][]TargetItem, allocator *Allocator) map[string]collectorJSON {
	displayData := make(map[string]collectorJSON)
	for _, j := range allocator.TargetItems() {
		if j.JobName == job {
			var targetList []TargetItem
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

func GetAllTargetsByCollectorAndJob(collector string, job string, cMap map[string][]TargetItem, allocator *Allocator) []targetGroupJSON {
	var tgs []targetGroupJSON
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
		tgs = append(tgs, targetGroupJSON{Targets: []string{v}, Labels: labelSet[v]})
	}

	return tgs
}
