// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	meterName = "crd-metrics"
)

// Metric labels

const (
	prefix               = "opentelemetry_collector_"
	receiversMetricName  = prefix + "receivers"
	exportersMetricName  = prefix + "exporters"
	processorsMetricName = prefix + "processors"
	extensionsMetricName = prefix + "extensions"
	connectorsMetricName = prefix + "connectors"
	modeMetricName       = prefix + "info"
)

// TODO: Refactor this logic, centralize it. See: https://github.com/open-telemetry/opentelemetry-operator/issues/2603
type componentDefinitions struct {
	receivers  []string
	processors []string
	exporters  []string
	extensions []string
	connectors []string
}

// Metrics hold all gauges for the different metrics related to the CRs
// +kubebuilder:object:generate=false
type Metrics struct {
	modeCounter       metric.Int64UpDownCounter
	receiversCounter  metric.Int64UpDownCounter
	exporterCounter   metric.Int64UpDownCounter
	processorCounter  metric.Int64UpDownCounter
	extensionsCounter metric.Int64UpDownCounter
	connectorsCounter metric.Int64UpDownCounter
}

// BootstrapMetrics configures the OpenTelemetry meter provider with the Prometheus exporter.
func BootstrapMetrics() (metric.MeterProvider, error) {
	exporter, err := prometheus.New(prometheus.WithRegisterer(metrics.Registry))
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter)), err
}

func NewMetrics(prv metric.MeterProvider, ctx context.Context, cl client.Reader) (*Metrics, error) {
	meter := prv.Meter(meterName)
	modeCounter, err := meter.Int64UpDownCounter(modeMetricName)
	if err != nil {
		return nil, err
	}
	receiversCounter, err := meter.Int64UpDownCounter(receiversMetricName)
	if err != nil {
		return nil, err
	}

	exporterCounter, err := meter.Int64UpDownCounter(exportersMetricName)
	if err != nil {
		return nil, err
	}

	processorCounter, err := meter.Int64UpDownCounter(processorsMetricName)
	if err != nil {
		return nil, err
	}

	extensionsCounter, err := meter.Int64UpDownCounter(extensionsMetricName)
	if err != nil {
		return nil, err
	}

	connectorsCounter, err := meter.Int64UpDownCounter(connectorsMetricName)
	if err != nil {
		return nil, err
	}

	m := &Metrics{
		modeCounter:       modeCounter,
		receiversCounter:  receiversCounter,
		exporterCounter:   exporterCounter,
		processorCounter:  processorCounter,
		extensionsCounter: extensionsCounter,
		connectorsCounter: connectorsCounter,
	}

	err = m.init(ctx, cl)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Init metrics from the first time the operator starts.
func (m *Metrics) init(ctx context.Context, cl client.Reader) error {
	list := &OpenTelemetryCollectorList{}
	if err := cl.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		m.create(ctx, &list.Items[i])
	}
	return nil
}

func (m *Metrics) create(ctx context.Context, collector *OpenTelemetryCollector) {
	m.updateComponentCounters(ctx, collector, true)
	m.updateGeneralCRMetricsComponents(ctx, collector, true)
}

func (m *Metrics) delete(ctx context.Context, collector *OpenTelemetryCollector) {
	m.updateComponentCounters(ctx, collector, false)
	m.updateGeneralCRMetricsComponents(ctx, collector, false)
}

func (m *Metrics) update(ctx context.Context, oldCollector *OpenTelemetryCollector, newCollector *OpenTelemetryCollector) {
	m.delete(ctx, oldCollector)
	m.create(ctx, newCollector)
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
	components := getComponentsFromConfig(collector.Spec.Config)
	moveCounter(ctx, collector, components.receivers, m.receiversCounter, up)
	moveCounter(ctx, collector, components.exporters, m.exporterCounter, up)
	moveCounter(ctx, collector, components.processors, m.processorCounter, up)
	moveCounter(ctx, collector, components.extensions, m.extensionsCounter, up)
	moveCounter(ctx, collector, components.connectors, m.connectorsCounter, up)

}

func extractElements(elements map[string]interface{}) []string {
	// TODO: we should get rid of this method and centralize the parse logic
	//		see https://github.com/open-telemetry/opentelemetry-operator/issues/2603
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

func getComponentsFromConfig(yamlContent Config) *componentDefinitions {

	info := &componentDefinitions{
		receivers: extractElements(yamlContent.Receivers.Object),
		exporters: extractElements(yamlContent.Exporters.Object),
	}

	if yamlContent.Processors != nil {
		info.processors = extractElements(yamlContent.Processors.Object)
	}

	if yamlContent.Extensions != nil {
		info.extensions = extractElements(yamlContent.Extensions.Object)
	}

	if yamlContent.Connectors != nil {
		info.connectors = extractElements(yamlContent.Connectors.Object)
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
