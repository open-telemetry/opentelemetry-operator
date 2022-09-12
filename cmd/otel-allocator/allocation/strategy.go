package allocation

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
)

type AllocatorProvider func(log logr.Logger) Allocator

var (
	registry = map[string]AllocatorProvider{}

	// TargetsPerCollector records how many targets have been assigned to each collector
	// It is currently the responsibility of the strategy to track this information.
	TargetsPerCollector = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_targets_per_collector",
		Help: "The number of targets for each collector.",
	}, []string{"collector_name", "strategy"})
	CollectorsAllocatable = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_collectors_allocatable",
		Help: "Number of collectors the allocator is able to allocate to.",
	}, []string{"strategy"})
	TimeToAssign = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "opentelemetry_allocator_time_to_allocate",
		Help: "The time it takes to allocate",
	}, []string{"method", "strategy"})
)

func New(name string, log logr.Logger) (Allocator, error) {
	if p, ok := registry[name]; ok {
		return p(log), nil
	}
	return nil, errors.New(fmt.Sprintf("unregistered strategy: %s", name))
}

func Register(name string, provider AllocatorProvider) error {
	if _, ok := registry[name]; ok {
		return errors.New("already registered")
	}
	registry[name] = provider
	return nil
}

type Allocator interface {
	SetCollectors(collectors map[string]*Collector)
	SetTargets(targets map[string]*TargetItem)
	TargetItems() map[string]*TargetItem
	Collectors() map[string]*Collector
}

type TargetItem struct {
	JobName       string
	Link          LinkJSON
	TargetURL     string
	Label         model.LabelSet
	CollectorName string
}

func NewTargetItem(jobName string, targetURL string, label model.LabelSet, collectorName string) *TargetItem {
	return &TargetItem{
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
