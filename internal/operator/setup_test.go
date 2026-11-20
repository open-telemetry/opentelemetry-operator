// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/apiserverendpoints"
)

func TestCacheByObjectWatchLabelSelector(t *testing.T) {
	byObject, err := cacheByObject("")
	require.NoError(t, err)
	require.Len(t, byObject, 1)
	endpointSlice, endpointSliceSettings := apiserverendpoints.CacheByObject()
	for object, settings := range byObject {
		require.Equal(t, reflect.TypeOf(endpointSlice), reflect.TypeOf(object))
		require.Equal(t, endpointSliceSettings, settings)
	}

	byObject, err = cacheByObject("operator.example.com/instance=blue")
	require.NoError(t, err)
	require.Len(t, byObject, 6)

	wantTypes := map[reflect.Type]bool{
		reflect.TypeFor[*v1beta1.OpenTelemetryCollector](): true,
		reflect.TypeFor[*v1alpha1.TargetAllocator]():       true,
		reflect.TypeFor[*v1alpha1.OpAMPBridge]():           true,
		reflect.TypeFor[*v1alpha1.Instrumentation]():       true,
		reflect.TypeFor[*v1alpha1.ClusterObservability]():  true,
	}
	for object, settings := range byObject {
		if wantTypes[reflect.TypeOf(object)] {
			require.True(t, settings.Label.Matches(labels.Set{"operator.example.com/instance": "blue"}))
			require.False(t, settings.Label.Matches(labels.Set{"operator.example.com/instance": "green"}))
			delete(wantTypes, reflect.TypeOf(object))
		} else {
			require.Equal(t, reflect.TypeOf(endpointSlice), reflect.TypeOf(object))
			require.Equal(t, endpointSliceSettings, settings)
		}
	}
	require.Empty(t, wantTypes)
}

func TestCacheByObjectRejectsInvalidWatchLabelSelector(t *testing.T) {
	_, err := cacheByObject("operator.example.com/instance in (")
	require.ErrorContains(t, err, "invalid watch-label-selector")
}
