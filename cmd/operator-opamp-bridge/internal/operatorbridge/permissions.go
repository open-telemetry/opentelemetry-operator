// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatorbridge

import bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"

// ListRequiredPermissions returns the Kubernetes permissions needed to run the bridge in operator mode.
func ListRequiredPermissions() ([]bridgemanager.Permission, error) {
	return []bridgemanager.Permission{
		{Verb: "get", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "list", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "create", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "update", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "delete", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "get", Resource: "pods"},
		{Verb: "list", Resource: "pods"},
	}, nil
}
