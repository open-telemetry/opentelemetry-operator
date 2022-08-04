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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

var _ AutoDetect = (*autoDetect)(nil)

// AutoDetect provides an assortment of routines that auto-detect traits based on the runtime.
type AutoDetect interface {
	Platform() (platform.Platform, error)
	HPAVersion() (string, error)
}

type autoDetect struct {
	dcl discovery.DiscoveryInterface
}

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

// Platform returns the detected platform this operator is running on. Possible values: Kubernetes, OpenShift.
func (a *autoDetect) Platform() (platform.Platform, error) {
	apiList, err := a.dcl.ServerGroups()
	if err != nil {
		return platform.Unknown, err
	}

	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "route.openshift.io" {
			return platform.OpenShift, nil
		}
	}

	return platform.Kubernetes, nil
}

func (a *autoDetect) HPAVersion() (string, error) {
	apiList, err := a.dcl.ServerGroups()
	if err != nil {
		return "", err
	}

	for _, apiGroup := range apiList.Groups {
		if apiGroup.Name == "autoscaling" {
			for _, version := range apiGroup.Versions {
				if version.Version == "v2" { // We can't use the constants from internal/config/main.go as that would create an import cycle
					return version.Version, nil
				}
			}
			return "v2beta2", nil
		}
	}

	return "", errors.New("Failed to find apiGroup autoscaling")
}
