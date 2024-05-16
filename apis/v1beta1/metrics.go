// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	meterName = "crd-metrics"
)

// Metric labels

const (
	prefix     = "opentelemetry_collector_"
	receivers  = prefix + "receivers"
	exporters  = prefix + "exporters"
	processors = prefix + "processors"
	extensions = prefix + "extensions"
	mode       = prefix + "info"
)

type components struct {
	receivers  []string
	processors []string
	exporters  []string
	extensions []string
}

// Metrics hold all gauges for the different metrics related to the CRs
// +kubebuilder:object:generate=false
type Metrics struct {
	modeCounter       metric.Int64UpDownCounter
	receiversCounter  metric.Int64UpDownCounter
	exporterCounter   metric.Int64UpDownCounter
	processorCounter  metric.Int64UpDownCounter
	extensionsCounter metric.Int64UpDownCounter
}

// BootstrapMetrics configures the OpenTelemetry meter provider with the Prometheus exporter.
func BootstrapMetrics() error {
	exporter, err := prometheus.New(prometheus.WithRegisterer(metrics.Registry))
	if err != nil {
		return err
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	return err
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(meterName)
	modeCounter, err := meter.Int64UpDownCounter(mode)
	if err != nil {
		return nil, err
	}
	receiversCounter, err := meter.Int64UpDownCounter(receivers)
	if err != nil {
		return nil, err
	}

	exporterCounter, err := meter.Int64UpDownCounter(exporters)
	if err != nil {
		return nil, err
	}

	processorCounter, err := meter.Int64UpDownCounter(processors)
	if err != nil {
		return nil, err
	}

	extensionsCounter, err := meter.Int64UpDownCounter(extensions)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		modeCounter:       modeCounter,
		receiversCounter:  receiversCounter,
		exporterCounter:   exporterCounter,
		processorCounter:  processorCounter,
		extensionsCounter: extensionsCounter,
	}, nil

}

func (m *Metrics) incCounters(ctx context.Context, collector *OpenTelemetryCollector) {
	m.updateComponentCounters(ctx, collector, true)
	m.updateGeneralCRMetricsComponents(ctx, collector, true)
}

func (m *Metrics) decCounters(ctx context.Context, collector *OpenTelemetryCollector) {
	m.updateComponentCounters(ctx, collector, false)
	m.updateGeneralCRMetricsComponents(ctx, collector, false)
}

func (m *Metrics) updateGeneralCRMetricsComponents(ctx context.Context, collector *OpenTelemetryCollector, up bool) {

	inc := 1
	if !up {
		inc = -1
	}
	m.modeCounter.Add(ctx, int64(inc), metric.WithAttributes(
		attribute.Key("collector_name").String(collector.Name),
		attribute.Key("namespace").String(collector.Namespace),
		attribute.Key("type").String(string(collector.Spec.Mode)),
	))
}
func (m *Metrics) updateComponentCounters(ctx context.Context, collector *OpenTelemetryCollector, up bool) {
	components := getComponentsFromConfigV1Beta1(collector.Spec.Config)
	moveCounter(ctx, collector, components.receivers, m.receiversCounter, up)
	moveCounter(ctx, collector, components.exporters, m.exporterCounter, up)
	moveCounter(ctx, collector, components.processors, m.processorCounter, up)
	moveCounter(ctx, collector, components.extensions, m.extensionsCounter, up)
}

func extractElements(elements map[string]interface{}) []string {
	if elements == nil {
		return []string{}
	}

	itemsMap := map[string]struct{}{}
	var items []string
	for key := range elements {
		itemName := strings.SplitN(key, "/", 2)[0]
		itemsMap[itemName] = struct{}{}
	}
	for key := range itemsMap {
		items = append(items, key)
	}
	return items
}

func getComponentsFromConfigV1Beta1(yamlContent Config) *components {

	info := &components{
		receivers: extractElements(yamlContent.Receivers.Object),
		exporters: extractElements(yamlContent.Exporters.Object),
	}

	if yamlContent.Processors != nil {
		info.processors = extractElements(yamlContent.Processors.Object)
	}

	if yamlContent.Extensions != nil {
		info.extensions = extractElements(yamlContent.Extensions.Object)
	}
	return info
}

func moveCounter(
	ctx context.Context, collector *OpenTelemetryCollector, types []string, upDown metric.Int64UpDownCounter, up bool) {
	for _, exporter := range types {
		inc := 1
		if !up {
			inc = -1
		}
		upDown.Add(ctx, int64(inc), metric.WithAttributes(
			attribute.Key("collector_name").String(collector.Name),
			attribute.Key("namespace").String(collector.Namespace),
			attribute.Key("type").String(exporter),
		))
	}
}
