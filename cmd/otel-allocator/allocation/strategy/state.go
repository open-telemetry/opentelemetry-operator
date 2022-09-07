package strategy

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

type TargetItem struct {
	JobName       string
	Link          LinkJSON
	TargetURL     string
	Label         model.LabelSet
	CollectorName string
}

func NewTargetItem(jobName string, targetURL string, label model.LabelSet, collectorName string) TargetItem {
	return TargetItem{
		JobName:       jobName,
		Link:          LinkJSON{fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(jobName))},
		TargetURL:     targetURL,
		Label:         label,
		CollectorName: collectorName,
	}
}

func (t TargetItem) Hash() string {
	return t.JobName + t.TargetURL + t.Label.Fingerprint().String()
}

// Collector Creates a struct that holds Collector information
// This struct will be parsed into endpoint with Collector and jobs info
// This struct can be extended with information like annotations and labels in the future
type Collector struct {
	Name       string
	NumTargets int
}

func NewCollector(name string) *Collector {
	return &Collector{Name: name}
}

type State struct {
	// collectors is a map from a Collector's name to a Collector instance
	collectors map[string]*Collector
	// targetItems is a map from a target item's hash to the target items allocated state
	targetItems map[string]*TargetItem
}

func (s State) Collectors() map[string]*Collector {
	return s.collectors
}

func (s State) TargetItems() map[string]*TargetItem {
	return s.targetItems
}

func (s State) SetTargetItem(key string, value *TargetItem) {
	s.targetItems[key] = value
}

func (s State) SetCollector(key string, value *Collector) {
	s.collectors[key] = value
}

func (s State) RemoveCollector(key string) {
	delete(s.collectors, key)
}

func (s State) RemoveTargetItem(key string) {
	delete(s.targetItems, key)
}

func NewState(collectors map[string]*Collector, targetItems map[string]*TargetItem) State {
	return State{collectors: collectors, targetItems: targetItems}
}
