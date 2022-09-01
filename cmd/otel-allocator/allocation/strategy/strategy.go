package strategy

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
)

type AllocatorProvider func() AllocatorStrategy

var (
	registry = map[string]AllocatorProvider{}

	// TargetsPerCollector records how many targets have been assigned to each collector
	// It is currently the responsibility of the strategy to track this information.
	TargetsPerCollector = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets_per_collector",
		Help: "The number of targets for each collector.",
	}, []string{"collector_name"})
)

func NewStrategy(name string) (AllocatorStrategy, error) {
	if p, ok := registry[name]; ok {
		return p(), nil
	}
	return nil, errors.New(fmt.Sprintf("unregistered strategy: %s", name))
}

func RegisterStrategy(name string, provider AllocatorProvider) error {
	if _, ok := registry[name]; ok {
		return errors.New("already registered")
	}
	registry[name] = provider
	return nil
}

type AllocatorStrategy interface {
	Allocate(currentState, newState State) State
}

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

// Create a struct that holds Collector - and jobs for that Collector
// This struct will be parsed into endpoint with Collector and jobs info
// This struct can be extended with information like annotations and labels in the future
type Collector struct {
	Name       string
	NumTargets int
}

type State struct {
	// collectors is a map from a Collector's name to a Collector instance
	collectors map[string]Collector
	// targetItems is a map from a target item's hash to the target items allocated state
	targetItems map[string]TargetItem
}

func (s State) Collectors() map[string]Collector {
	return s.collectors
}

func (s State) TargetItems() map[string]TargetItem {
	return s.targetItems
}

func NewState(collectors map[string]Collector, targetItems map[string]TargetItem) State {
	return State{collectors: collectors, targetItems: targetItems}
}
