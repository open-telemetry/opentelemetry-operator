package allocation

import (
	"fmt"
	"net/url"

	"github.com/prometheus/common/model"
)

type LinkJSON struct {
	Link string `json:"_link"`
}

type CollectorJSON struct {
	Link string            `json:"_link"`
	Jobs []TargetGroupJSON `json:"targets"`
}

type TargetGroupJSON struct {
	Targets []string       `json:"targets"`
	Labels  model.LabelSet `json:"labels"`
}

func GetAllTargetsByJob(job string, cMap map[string][]TargetItem, allocator Allocator) map[string]CollectorJSON {
	displayData := make(map[string]CollectorJSON)
	for _, j := range allocator.TargetItems() {
		if j.JobName == job {
			var targetList []TargetItem
			targetList = append(targetList, cMap[j.CollectorName+j.JobName]...)

			var targetGroupList []TargetGroupJSON

			for _, t := range targetList {
				targetGroupList = append(targetGroupList, TargetGroupJSON{
					Targets: []string{t.TargetURL},
					Labels:  t.Label,
				})
			}

			displayData[j.CollectorName] = CollectorJSON{Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", url.QueryEscape(j.JobName), j.CollectorName), Jobs: targetGroupList}

		}
	}
	return displayData
}

func GetAllTargetsByCollectorAndJob(collector string, job string, cMap map[string][]TargetItem, allocator Allocator) []TargetGroupJSON {
	var tgs []TargetGroupJSON
	group := make(map[string]string)
	labelSet := make(map[string]model.LabelSet)
	if _, ok := allocator.Collectors()[collector]; ok {
		for _, targetItemArr := range cMap {
			for _, targetItem := range targetItemArr {
				if targetItem.CollectorName == collector && targetItem.JobName == job {
					group[targetItem.Label.String()] = targetItem.TargetURL
					labelSet[targetItem.TargetURL] = targetItem.Label
				}
			}
		}
	}
	for _, v := range group {
		tgs = append(tgs, TargetGroupJSON{Targets: []string{v}, Labels: labelSet[v]})
	}

	return tgs
}
