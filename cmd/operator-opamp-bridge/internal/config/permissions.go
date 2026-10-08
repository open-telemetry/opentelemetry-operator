// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import "fmt"

// Permission describes a single Kubernetes access the bridge must be allowed to perform.
type Permission struct {
	Verb      string
	APIGroup  string
	Resource  string
	Namespace string
	Name      string
}

// Description renders the permission target for error messages.
func (p Permission) Description() string {
	resource := p.Resource
	if p.APIGroup != "" {
		resource = p.APIGroup + "/" + resource
	}
	if p.Namespace == "" && p.Name == "" {
		return resource
	}
	if p.Name == "" {
		return fmt.Sprintf("%s in namespace %s", resource, p.Namespace)
	}
	return fmt.Sprintf("%s %s/%s", resource, p.Namespace, p.Name)
}

// OperatorPermissions returns the Kubernetes permissions needed to run the bridge in operator mode.
// Patch permissions on workloads are only required when the AcceptsRestartCommand capability is enabled
// because the restart command triggers rollouts.
func (c *Config) OperatorPermissions() ([]Permission, error) {
	perms := []Permission{
		{Verb: "get", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "list", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "create", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "update", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "delete", APIGroup: "opentelemetry.io", Resource: "opentelemetrycollectors"},
		{Verb: "get", Resource: "pods"},
		{Verb: "list", Resource: "pods"},
		{Verb: "get", APIGroup: "apps", Resource: "deployments"},
		{Verb: "get", APIGroup: "apps", Resource: "daemonsets"},
		{Verb: "get", APIGroup: "apps", Resource: "statefulsets"},
	}
	if c.RestartCommandEnabled() {
		perms = append(perms,
			Permission{Verb: "patch", APIGroup: "apps", Resource: "deployments"},
			Permission{Verb: "patch", APIGroup: "apps", Resource: "daemonsets"},
			Permission{Verb: "patch", APIGroup: "apps", Resource: "statefulsets"},
		)
	}
	return perms, nil
}
