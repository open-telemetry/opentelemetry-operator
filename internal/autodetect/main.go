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

// Package autodetect is for auto-detecting traits from the environment (platform, APIs, ...).
package autodetect

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

var _ AutoDetect = (*autoDetect)(nil)

// AutoDetect provides an assortment of routines that auto-detect traits based on the runtime.
type AutoDetect interface {
	OpenShiftRoutesAvailability() (openshift.RoutesAvailability, error)
	PrometheusCRsAvailability() (prometheus.Availability, error)
	RBACPermissions(ctx context.Context) (autoRBAC.Availability, error)
}

type autoDetect struct {
	dcl      discovery.DiscoveryInterface
	reviewer *rbac.Reviewer
}

// New creates a new auto-detection worker, using the given client when talking to the current cluster.
func New(restConfig *rest.Config, reviewer *rbac.Reviewer) (AutoDetect, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		// it's pretty much impossible to get into this problem, as most of the
		// code branches from the previous call just won't fail at all,
		// but let's handle this error anyway...
		return nil, err
	}

	return &autoDetect{
		dcl:      dcl,
		reviewer: reviewer,
	}, nil
}

// PrometheusCRsAvailability checks if Prometheus CRDs are available.
func (a *autoDetect) PrometheusCRsAvailability() (prometheus.Availability, error) {
	apiList, err := a.dcl.ServerGroups()
	if err != nil {
		return prometheus.NotAvailable, err
	}

	foundServiceMonitor := false
	foundPodMonitor := false
	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "monitoring.coreos.com" {
			for _, version := range apiGroups[i].Versions {
				resources, err := a.dcl.ServerResourcesForGroupVersion(version.GroupVersion)
				if err != nil {
					return prometheus.NotAvailable, err
				}

				for _, resource := range resources.APIResources {
					if resource.Kind == "ServiceMonitor" {
						foundServiceMonitor = true
					} else if resource.Kind == "PodMonitor" {
						foundPodMonitor = true
					}
				}
			}
		}
	}

	if foundServiceMonitor && foundPodMonitor {
		return prometheus.Available, nil
	}

	return prometheus.NotAvailable, nil
}

// OpenShiftRoutesAvailability checks if OpenShift Route are available.
func (a *autoDetect) OpenShiftRoutesAvailability() (openshift.RoutesAvailability, error) {
	apiList, err := a.dcl.ServerGroups()
	if err != nil {
		return openshift.RoutesNotAvailable, err
	}

	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "route.openshift.io" {
			return openshift.RoutesAvailable, nil
		}
	}

	return openshift.RoutesNotAvailable, nil
}

func (a *autoDetect) RBACPermissions(ctx context.Context) (autoRBAC.Availability, error) {
	w, err := autoRBAC.CheckRBACPermissions(ctx, a.reviewer)
	if err != nil {
		return autoRBAC.NotAvailable, err
	}
	if w != nil {
		return autoRBAC.NotAvailable, fmt.Errorf("missing permissions: %s", w)
	}

	return autoRBAC.Available, nil
}
