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

package instrumentation

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/strings/slices"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

var defaultSize = resource.MustParse("150Mi")

// Calculate if we already inject InitContainers.
func isInitContainerMissing(pod corev1.Pod, containerName string) bool {
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == containerName {
			return false
		}
	}
	return true
}

// Checks if Pod is already instrumented by checking Instrumentation InitContainer presence.
func isAutoInstrumentationInjected(pod corev1.Pod) bool {
	for _, cont := range pod.Spec.InitContainers {
		if slices.Contains([]string{dotnetInitContainerName, javaInitContainerName,
			nodejsInitContainerName, pythonInitContainerName, apacheAgentInitContainerName}, cont.Name) {
			return true
		}
	}

	for _, cont := range pod.Spec.Containers {
		// Go uses a sidecar
		if cont.Name == sideCarName {
			return true
		}

		// This environment variable is set in the sidecar and in the
		// collector containers. We look for it in any container that is not
		// the sidecar container to check if we already injected the
		// instrumentation or not
		if cont.Name != naming.Container() {
			for _, envVar := range cont.Env {
				if envVar.Name == constants.EnvNodeName {
					return true
				}
			}
		}
	}
	return false
}

// Look for duplicates in the provided containers.
func findDuplicatedContainers(ctrs []string) error {
	// Merge is needed because of multiple containers can be provided for single instrumentation.
	mergedContainers := strings.Join(ctrs, ",")

	// Split all containers.
	splitContainers := strings.Split(mergedContainers, ",")

	countMap := make(map[string]int)
	var duplicates []string
	for _, str := range splitContainers {
		countMap[str]++
	}

	// Find and collect the duplicates
	for str, count := range countMap {
		// omit empty container names
		if str == "" {
			continue
		}

		if count > 1 {
			duplicates = append(duplicates, str)
		}
	}

	if duplicates != nil {
		sort.Strings(duplicates)
		return fmt.Errorf("duplicated container names detected: %s", duplicates)
	}

	return nil
}

// Return positive for instrumentation with defined containers.
func isInstrWithContainers(inst instrumentationWithContainers) int {
	if inst.Containers != "" {
		return 1
	}

	return 0
}

// Return positive for instrumentation without defined containers.
func isInstrWithoutContainers(inst instrumentationWithContainers) int {
	if inst.Containers == "" {
		return 1
	}

	return 0
}

func volumeSize(quantity *resource.Quantity) *resource.Quantity {
	if quantity == nil {
		return &defaultSize
	}
	return quantity
}
