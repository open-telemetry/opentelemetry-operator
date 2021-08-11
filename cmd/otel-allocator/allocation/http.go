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

func (allocator *Allocator) getAllTargetsByJob(job string, cMap map[string][]TargetItem) map[string]collectorJSON {
	displayData := make(map[string]collectorJSON)
	for _, j := range allocator.targetItems {
		if j.JobName == job {
			var jobsArr []TargetItem
			jobsArr = append(jobsArr, cMap[j.Collector.Name+j.JobName]...)

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
			displayData[j.Collector.Name] = collectorJSON{Link: "/jobs/" + j.JobName + "/targets" + "?collector_id=" + j.Collector.Name, Jobs: targetGroupList}
		}
	}
	return displayData
}

func (allocator *Allocator) getAllTargetsByCollectorAndJob(collector string, job string, cMap map[string][]TargetItem) []targetGroupJSON {
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

func (allocator *Allocator) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]

	var compareMap = make(map[string][]TargetItem) // CollectorName+jobName -> TargetItem
	for _, v := range allocator.targetItems {
		compareMap[v.Collector.Name+v.JobName] = append(compareMap[v.Collector.Name+v.JobName], *v)
	}
	params := mux.Vars(r)

	if len(q) == 0 {
		displayData := allocator.getAllTargetsByJob(params["job_id"], compareMap)
		allocator.jsonHandler(w, r, displayData)

	} else {
		tgs := allocator.getAllTargetsByCollectorAndJob(q[0], params["job_id"], compareMap)
		// Displays empty list if nothing matches
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
