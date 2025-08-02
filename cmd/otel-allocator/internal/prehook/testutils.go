// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
)

func CompareTargetsMap(a, b map[target.ItemHash]*target.Item) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if v2, ok := b[k]; !ok || !reflect.DeepEqual(v, v2) {
			return false
		}
	}

	return true
}

func MakeTargetFromProm(rCfgs []*relabel.Config, rawTarget *target.Item) (*target.Item, error) {
	lb := labels.NewBuilder(rawTarget.Labels)
	cfg := &config.ScrapeConfig{
		RelabelConfigs: rCfgs,
	}
	lset, origLabels, err := PopulateLabels(lb, cfg)
	if err != nil {
		return nil, err
	}
	// If the lset is empty after relabeling, Prometheus drops the target.
	if lset.IsEmpty() {
		return nil, nil
	}

	newTarget := target.NewItem(rawTarget.JobName, lset.Get(model.AddressLabel), lset, rawTarget.CollectorName, target.WithReservedLabelMatching(origLabels))
	return newTarget, nil
}

// PopulateLabels is Copied from prometheus/scrape/target.go.
// Reason: "github.com/prometheus/common@0.65.0" and "github.com/prometheus/scrape@0.301.0" are incompatible (undefined: promslog.AllowedFormat).
func PopulateLabels(lb *labels.Builder, cfg *config.ScrapeConfig) (res, orig labels.Labels, err error) {
	// Copy labels into the labelset for the target if they are not set already.
	scrapeLabels := []labels.Label{
		{Name: model.JobLabel, Value: cfg.JobName},
		{Name: model.ScrapeIntervalLabel, Value: cfg.ScrapeInterval.String()},
		{Name: model.ScrapeTimeoutLabel, Value: cfg.ScrapeTimeout.String()},
		{Name: model.MetricsPathLabel, Value: cfg.MetricsPath},
		{Name: model.SchemeLabel, Value: cfg.Scheme},
	}

	for _, l := range scrapeLabels {
		if lb.Get(l.Name) == "" {
			lb.Set(l.Name, l.Value)
		}
	}
	// Encode scrape query parameters as labels.
	for k, v := range cfg.Params {
		if name := model.ParamLabelPrefix + k; len(v) > 0 && lb.Get(name) == "" {
			lb.Set(name, v[0])
		}
	}

	preRelabelLabels := lb.Labels()
	keep := relabel.ProcessBuilder(lb, cfg.RelabelConfigs...)

	// Check if the target was dropped.
	if !keep {
		return labels.EmptyLabels(), preRelabelLabels, nil
	}
	if v := lb.Get(model.AddressLabel); v == "" {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("no address")
	}

	addr := lb.Get(model.AddressLabel)

	if err := config.CheckTargetAddress(model.LabelValue(addr)); err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), err
	}

	interval := lb.Get(model.ScrapeIntervalLabel)
	intervalDuration, err := model.ParseDuration(interval)
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("error parsing scrape interval: %w", err)
	}
	if time.Duration(intervalDuration) == 0 {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("scrape interval cannot be 0")
	}

	timeout := lb.Get(model.ScrapeTimeoutLabel)
	timeoutDuration, err := model.ParseDuration(timeout)
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("error parsing scrape timeout: %w", err)
	}
	if time.Duration(timeoutDuration) == 0 {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("scrape timeout cannot be 0")
	}

	if timeoutDuration > intervalDuration {
		return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("scrape timeout cannot be greater than scrape interval (%q > %q)", timeout, interval)
	}

	// Meta labels are deleted after relabelling. Other internal labels propagate to
	// the target which decides whether they will be part of their label set.
	lb.Range(func(l labels.Label) {
		if strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
			lb.Del(l.Name)
		}
	})

	// Default the instance label to the target address.
	if v := lb.Get(model.InstanceLabel); v == "" {
		lb.Set(model.InstanceLabel, addr)
	}

	res = lb.Labels()
	err = res.Validate(func(l labels.Label) error {
		// Check label values are valid, drop the target if not.
		if !model.LabelValue(l.Value).IsValid() {
			return fmt.Errorf("invalid label value for %q: %q", l.Name, l.Value)
		}
		return nil
	})
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), err
	}
	return res, preRelabelLabels, nil
}
