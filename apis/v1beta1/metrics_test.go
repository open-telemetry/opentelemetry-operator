// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var wantInstrumentationScope = instrumentation.Scope{
	Name: "crd-metrics",
}

func TestOTELCollectorCRDMetrics(t *testing.T) {

	otelcollector1 := &OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector1",
			Namespace: "test1",
		},
		Spec: OpenTelemetryCollectorSpec{
			Mode: ModeDeployment,
			Config: Config{
				Processors: &AnyConfig{
					Object: map[string]interface{}{
						"batch": nil,
						"foo":   nil,
					},
				},
				Extensions: &AnyConfig{
					Object: map[string]interface{}{
						"extfoo": nil,
					},
				},
			},
		},
	}

	otelcollector2 := &OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector2",
			Namespace: "test2",
		},
		Spec: OpenTelemetryCollectorSpec{
			Mode: ModeSidecar,
			Config: Config{
				Processors: &AnyConfig{
					Object: map[string]interface{}{
						"x": nil,
						"y": nil,
					},
				},
				Extensions: &AnyConfig{
					Object: map[string]interface{}{
						"z/r": nil,
					},
				},
				Exporters: AnyConfig{
					Object: map[string]interface{}{
						"w": nil,
					},
				},
			},
		},
	}

	updatedCollector1 := &OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector1",
			Namespace: "test1",
		},
		Spec: OpenTelemetryCollectorSpec{
			Mode: ModeSidecar,
			Config: Config{
				Processors: &AnyConfig{
					Object: map[string]interface{}{
						"foo": nil,
						"y":   nil,
					},
				},
				Extensions: &AnyConfig{
					Object: map[string]interface{}{
						"z/r": nil,
					},
				},
				Exporters: AnyConfig{
					Object: map[string]interface{}{
						"w": nil,
					},
				},
			},
		},
	}

	var tests = []struct {
		name         string
		testFunction func(t *testing.T, m *Metrics, collectors []*OpenTelemetryCollector, reader metric.Reader)
	}{
		{
			name:         "create",
			testFunction: checkCreate,
		},
		{
			name:         "update",
			testFunction: checkUpdate,
		},
		{
			name:         "delete",
			testFunction: checkDelete,
		},
	}
	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
		metav1.AddToGroupVersion(s, GroupVersion)
		return nil
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err)
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	crdMetrics, err := NewMetrics(provider, context.Background(), cl)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunction(t, crdMetrics, []*OpenTelemetryCollector{otelcollector1, otelcollector2, updatedCollector1}, reader)
		})
	}
}

func TestOTELCollectorInitMetrics(t *testing.T) {
	otelcollector1 := OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector1",
			Namespace: "test1",
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "opentelemetry-operator"},
		},
		Spec: OpenTelemetryCollectorSpec{
			Mode: ModeDeployment,
			Config: Config{
				Processors: &AnyConfig{
					Object: map[string]interface{}{
						"batch": nil,
						"foo":   nil,
					},
				},
				Extensions: &AnyConfig{
					Object: map[string]interface{}{
						"extfoo": nil,
					},
				},
			},
		},
	}

	otelcollector2 := OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector2",
			Namespace: "test2",
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "opentelemetry-operator"},
		},
		Spec: OpenTelemetryCollectorSpec{
			Mode: ModeSidecar,
			Config: Config{
				Processors: &AnyConfig{
					Object: map[string]interface{}{
						"x": nil,
						"y": nil,
					},
				},
				Extensions: &AnyConfig{
					Object: map[string]interface{}{
						"z/r": nil,
					},
				},
				Exporters: AnyConfig{
					Object: map[string]interface{}{
						"w": nil,
					},
				},
			},
		},
	}

	schemeBuilder := runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
		metav1.AddToGroupVersion(s, GroupVersion)
		return nil
	})
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	require.NoError(t, err)
	list := &OpenTelemetryCollectorList{
		Items: []OpenTelemetryCollector{otelcollector1, otelcollector2},
	}
	require.NoError(t, err, "Should be able to add custom types")
	cl := fake.NewClientBuilder().WithLists(list).WithScheme(scheme).Build()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	_, err = NewMetrics(provider, context.Background(), cl)
	assert.NoError(t, err)

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name: "opentelemetry_collector_info",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("deployment"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String(string(ModeSidecar)),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_processors",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("batch"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("foo"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("x"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("y"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_extensions",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("extfoo"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("z"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_exporters",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("w"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp())
}

func checkCreate(t *testing.T, m *Metrics, collectors []*OpenTelemetryCollector, reader metric.Reader) {
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	m.create(context.Background(), collectors[0])
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)

	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name: "opentelemetry_collector_info",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("deployment"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_processors",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("batch"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("foo"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_extensions",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("extfoo"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
	}
	require.Len(t, rm.ScopeMetrics, 1)
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp())

	m.create(context.Background(), collectors[1])

	rm = metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	want = metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name: "opentelemetry_collector_info",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("deployment"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String(string(ModeSidecar)),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_processors",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("batch"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("foo"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("x"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("y"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_extensions",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("extfoo"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("z"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_exporters",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("w"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp())
}

func checkUpdate(t *testing.T, m *Metrics, collectors []*OpenTelemetryCollector, reader metric.Reader) {

	m.update(context.Background(), collectors[0], collectors[2])

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name: "opentelemetry_collector_info",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String(string(ModeDeployment)),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String(string(ModeSidecar)),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String(string(ModeSidecar)),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_processors",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("batch"),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("foo"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("y"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("x"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("y"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_extensions",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("extfoo"),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("z"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("z"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_exporters",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("w"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("w"),
							),
							Value: 1,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp())
}

func checkDelete(t *testing.T, m *Metrics, collectors []*OpenTelemetryCollector, reader metric.Reader) {
	m.delete(context.Background(), collectors[1])
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	want := metricdata.ScopeMetrics{
		Scope: wantInstrumentationScope,
		Metrics: []metricdata.Metrics{
			{
				Name: "opentelemetry_collector_info",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String(string(ModeDeployment)),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String(string(ModeSidecar)),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String(string(ModeSidecar)),
							),
							Value: 0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_processors",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("batch"),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("foo"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("y"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("x"),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("y"),
							),
							Value: 0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_extensions",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("extfoo"),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("z"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("z"),
							),
							Value: 0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
			{
				Name: "opentelemetry_collector_exporters",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector1"),
								attribute.Key("namespace").String("test1"),
								attribute.Key("type").String("w"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								attribute.Key("collector_name").String("collector2"),
								attribute.Key("namespace").String("test2"),
								attribute.Key("type").String("w"),
							),
							Value: 0,
						},
					},
					Temporality: metricdata.CumulativeTemporality,
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp())
}
