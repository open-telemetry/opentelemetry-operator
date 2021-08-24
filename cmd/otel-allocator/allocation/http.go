package allocation

import (
	"fmt"

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
	for _, j := range allocator.TargetItems {
		if j.JobName == job {
			var targetList []TargetItem
			targetList = append(targetList, cMap[j.Collector.Name+j.JobName]...)

			var targetGroupList []targetGroupJSON

			trg := make(map[string][]TargetItem)
			for _, t := range targetList {
				trg[t.JobName+t.Label.String()] = append(trg[t.JobName+t.Label.String()], t)
			}
			labelSetMap := make(map[string]model.LabelSet)
			for _, tArr := range trg {
				var targets []string
				for _, t := range tArr {
					labelSetMap[t.TargetURL] = t.Label
					targets = append(targets, t.TargetURL)
				}
				targetGroupList = append(targetGroupList, targetGroupJSON{Targets: targets, Labels: labelSetMap[targets[0]]})

			}
			displayData[j.Collector.Name] = collectorJSON{Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", j.JobName, j.Collector.Name), Jobs: targetGroupList}

		}
	}
	return displayData
}

func GetAllTargetsByCollectorAndJob(collector string, job string, cMap map[string][]TargetItem, allocator *Allocator) []targetGroupJSON {
	var tgs []targetGroupJSON
	group := make(map[string][]string)
	labelSet := make(map[string]model.LabelSet)
	for _, col := range allocator.collectors {
		if col.Name == collector {
			for _, targetItemArr := range cMap {
				for _, targetItem := range targetItemArr {
					if targetItem.Collector.Name == collector && targetItem.JobName == job {
						group[targetItem.Label.String()] = append(group[targetItem.Label.String()], targetItem.TargetURL)
						labelSet[targetItem.TargetURL] = targetItem.Label
					}
				}
			}
		}
	}

	for _, v := range group {
		tgs = append(tgs, targetGroupJSON{Targets: v, Labels: labelSet[v[0]]})
	}

	return tgs
}
