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
	"errors"
	"sort"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var _ AutoDetect = (*autoDetect)(nil)

// AutoDetect provides an assortment of routines that auto-detect traits based on the runtime.
type AutoDetect interface {
	OpenShiftRoutesAvailability() (OpenShiftRoutesAvailability, error)
	HPAVersion() (AutoscalingVersion, error)
}

type autoDetect struct {
	dcl discovery.DiscoveryInterface
}

type AutoscalingVersion int

const (
	AutoscalingVersionV2 AutoscalingVersion = iota
	AutoscalingVersionV2Beta2
	AutoscalingVersionUnknown
)

const DefaultAutoscalingVersion = AutoscalingVersionV2

// New creates a new auto-detection worker, using the given client when talking to the current cluster.
func New(restConfig *rest.Config) (AutoDetect, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		// it's pretty much impossible to get into this problem, as most of the
		// code branches from the previous call just won't fail at all,
		// but let's handle this error anyway...
		return nil, err
	}

	return &autoDetect{
		dcl: dcl,
	}, nil
}

// OpenShiftRoutesAvailability checks if OpenShift Route are available.
func (a *autoDetect) OpenShiftRoutesAvailability() (OpenShiftRoutesAvailability, error) {
	apiList, err := a.dcl.ServerGroups()
	if err != nil {
		return OpenShiftRoutesNotAvailable, err
	}

	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "route.openshift.io" {
			return OpenShiftRoutesAvailable, nil
		}
	}

	return OpenShiftRoutesNotAvailable, nil
}

func (a *autoDetect) HPAVersion() (AutoscalingVersion, error) {
	apiList, err := a.dcl.ServerGroups()
	if err != nil {
		return AutoscalingVersionUnknown, err
	}

	for _, apiGroup := range apiList.Groups {
		if apiGroup.Name == "autoscaling" {
			// Sort this so we can make sure to get v2 before v2beta2
			versions := apiGroup.Versions
			sort.Slice(versions, func(i, j int) bool {
				return versions[i].Version < versions[j].Version
			})

			for _, version := range versions {
				if version.Version == "v2" || version.Version == "v2beta2" {
					return ToAutoScalingVersion(version.Version), nil
				}
			}
			return AutoscalingVersionUnknown, errors.New("Failed to find appropriate version of apiGroup autoscaling, only v2 and v2beta2 are supported")
		}
	}

	return AutoscalingVersionUnknown, errors.New("Failed to find apiGroup autoscaling")
}

func (v AutoscalingVersion) String() string {
	switch v {
	case AutoscalingVersionV2:
		return "v2"
	case AutoscalingVersionV2Beta2:
		return "v2beta2"
	case AutoscalingVersionUnknown:
		return "unknown"
	}
	return "unknown"
}

func ToAutoScalingVersion(version string) AutoscalingVersion {
	switch version {
	case "v2":
		return AutoscalingVersionV2
	case "v2beta2":
		return AutoscalingVersionV2Beta2
	}
	return AutoscalingVersionUnknown
}
