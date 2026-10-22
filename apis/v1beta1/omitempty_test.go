// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// TestOmitemptyActuallyOmits verifies that struct fields tagged with
// json:",omitempty" are actually omitted when zero-valued. Go's encoding/json
// never omits a zero-value struct — only pointers, slices, maps, and
// primitives. A non-pointer struct field with omitempty is always serialized,
// which breaks server-side apply when the serialized JSON contains fields that
// do not exist in an older CRD version on the cluster.
//
// This test marshals zero-valued specs to JSON and checks that omitempty fields
// do not appear in the output. Pre-existing violations are tracked by count;
// when you fix one, lower knownViolations so no new ones can be introduced.
func TestOmitemptyActuallyOmits(t *testing.T) {
	roots := []reflect.Type{
		reflect.TypeFor[v1beta1.OpenTelemetryCollector](),
		reflect.TypeFor[v1beta1.Instrumentation](),
		reflect.TypeFor[v1alpha1.TargetAllocator](),
		reflect.TypeFor[v1alpha1.OpAMPBridge](),
		reflect.TypeFor[v1alpha1.OpenTelemetryCollector](),
		reflect.TypeFor[v1alpha1.Instrumentation](),
		reflect.TypeFor[v1alpha1.ClusterObservability](),
	}

	seen := map[reflect.Type]bool{}
	var violations []string

	for _, root := range roots {
		checkOmitemptyOmits(root, seen, &violations)
	}
	slices.Sort(violations)

	// Pre-existing violations. Lower this number when fixing one; the test
	// fails if violations increase (new non-pointer struct with omitempty) or
	// decrease without updating (fixed but forgot to update the count).
	const knownViolations = 84

	if len(violations) > knownViolations {
		t.Errorf("found %d omitempty violations, expected at most %d; new struct fields with omitempty must use a pointer type",
			len(violations), knownViolations)
		for _, v := range violations {
			t.Error(v)
		}
	}
	if len(violations) < knownViolations {
		t.Errorf("violation count dropped from %d to %d — update knownViolations in this test",
			knownViolations, len(violations))
	}
}

// checkOmitemptyOmits marshals a zero-valued instance of typ to JSON and
// reports any field tagged omitempty that still appears in the output. It
// recurses into struct fields reachable from our API types.
func checkOmitemptyOmits(typ reflect.Type, seen map[reflect.Type]bool, violations *[]string) {
	typ = deref(typ)
	if typ.Kind() != reflect.Struct || seen[typ] {
		return
	}
	seen[typ] = true

	ours := strings.HasPrefix(typ.PkgPath(), "github.com/open-telemetry/opentelemetry-operator/apis")

	if ours {
		zero := reflect.New(typ).Interface()
		data, err := json.Marshal(zero)
		if err == nil {
			var m map[string]any
			if json.Unmarshal(data, &m) == nil {
				checkFields(typ, m, violations)
			}
		}
	}

	for field := range typ.Fields() {
		ft := elemType(field.Type)
		if ft.Kind() == reflect.Struct {
			checkOmitemptyOmits(ft, seen, violations)
		}
	}
}

// checkFields checks each field of typ: if it has omitempty and its json key
// still appears in the marshaled zero-value map, that's a violation.
func checkFields(typ reflect.Type, m map[string]any, violations *[]string) {
	for field := range typ.Fields() {
		if field.Anonymous {
			continue
		}
		tag := field.Tag.Get("json")
		if !jsonTagHasOmitempty(tag) {
			continue
		}
		jsonKey, _, _ := strings.Cut(tag, ",")
		if jsonKey == "" || jsonKey == "-" {
			continue
		}
		if _, found := m[jsonKey]; found {
			*violations = append(*violations, fmt.Sprintf(
				"%s.%s.%s: omitempty field %q present in zero-value JSON (type %s); use a pointer",
				path.Base(typ.PkgPath()), typ.Name(), field.Name, jsonKey, field.Type,
			))
		}
	}
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func elemType(t reflect.Type) reflect.Type {
	for {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			t = t.Elem()
		default:
			return t
		}
	}
}

func jsonTagHasOmitempty(tag string) bool {
	return slices.Contains(strings.Split(tag, ","), "omitempty")
}
