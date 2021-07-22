package allocation

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/common/model"
)

type TargetItem struct {
	JobName   string
	Link      linkJSON
	TargetURL string
	Label     model.LabelSet
	Collector *collector
}

type linkJSON struct {
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

func (allocator *Allocator) JobHandler(w http.ResponseWriter, r *http.Request) {
	displayData := make(map[string]linkJSON)
	for _, v := range allocator.targetItems {
		displayData[v.JobName] = linkJSON{v.Link.Link}
	}
	allocator.jsonHandler(w, r, displayData)
}

func (allocator *Allocator) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]

	var compareMap = make(map[string][]TargetItem) // CollectorName+jobName -> TargetItem
	for _, v := range allocator.targetItems {
		compareMap[v.Collector.Name+v.JobName] = append(compareMap[v.Collector.Name+v.JobName], *v)
	}
	displayData := make(map[string]collectorJSON)
	params := mux.Vars(r)

	if len(q) == 0 {
		for _, job := range allocator.targetItems {
			if job.JobName == params["job_id"] {
				var jobsArr []TargetItem
				jobsArr = append(jobsArr, compareMap[job.Collector.Name+job.JobName]...)

				var targetGroupList []targetGroupJSON

				trg := make(map[string][]TargetItem)
				for _, t := range jobsArr {
					trg[t.JobName+t.Label.String()] = append(trg[t.JobName+t.Label.String()], t)
				}
				labelSetMap := make(map[string]model.LabelSet)
				for _, tArr := range trg {
					var targetArr []string
					for _, t := range tArr {
						labelSetMap[t.TargetURL] = t.Label
						targetArr = append(targetArr, t.TargetURL)
					}
					targetGroupList = append(targetGroupList, targetGroupJSON{Targets: targetArr, Labels: labelSetMap[targetArr[0]]})

				}
				displayData[job.Collector.Name] = collectorJSON{Link: "/jobs/" + job.JobName + "/targets" + "?collector_id=" + job.Collector.Name, Jobs: targetGroupList}
			}
		}
		allocator.jsonHandler(w, r, displayData)

	} else {
		var tgs []targetGroupJSON
		group := make(map[string][]string)
		labelSet := make(map[string]model.LabelSet)
		for _, col := range allocator.collectors {
			if col.Name == q[0] {
				for _, targetItemArr := range compareMap {
					for _, targetItem := range targetItemArr {
						if targetItem.Collector.Name == q[0] && targetItem.JobName == params["job_id"] {
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
		if len(tgs) == 0 {
			allocator.jsonHandler(w, r, []interface{}{})
			return
		}
		allocator.jsonHandler(w, r, tgs)
	}
}

func (s *Allocator) jsonHandler(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
