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
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
)

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
	// Go uses a side car
	for _, cont := range pod.Spec.Containers {
		if cont.Name == sideCarName {
			return true
		}
	}
	return false
}

// Check if single instrumentation is configured for Pod and return which is configured.
func isSingleInstrumentationEnabled(insts languageInstrumentations) (bool, string) {
	// Check if more than one field is not nil
	count := 0
	enabledInstrumentation := ""
	value := reflect.ValueOf(insts)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i).FieldByName("Instrumentation")
		if !field.IsNil() {
			count++
			enabledInstrumentation = value.Type().Field(i).Name
		}
	}

	if count == 1 {
		return true, enabledInstrumentation
	} else {
		return false, ""
	}
}

// Check if specific containers are provided for configured instrumentation.
func areContainerNamesConfiguredForMultipleInstrumentations(langInsts languageInstrumentations) (bool, string) {
	instrWithoutContainers := 0
	instrWithContainers := 0
	var allContainers []string

	insts := reflect.ValueOf(langInsts)
	for i := 0; i < insts.NumField(); i++ {
		language := insts.Field(i)
		instr := language.FieldByName("Instrumentation")
		containers := language.FieldByName("Containers")

		if !instr.IsNil() && containers.String() == "" {
			instrWithoutContainers++
		}

		if !instr.IsNil() && containers.String() != "" {
			instrWithContainers++
		}

		allContainers = append(allContainers, containers.String())
	}

	// Look for duplicated containers.
	containerDuplicates := findDuplicatedContainers(allContainers)
	if containerDuplicates != nil {
		return false, fmt.Sprintf("duplicated container names detected: %s", containerDuplicates)
	}

	// Look for mixed multiple instrumentations with and without container names.
	if instrWithoutContainers > 0 && instrWithContainers > 0 {
		return false, "incorrect instrumentation configuration - please provide container names for all instrumentations"
		// Look for multiple instrumentations without container names.
	} else if instrWithoutContainers > 1 && instrWithContainers == 0 {
		return false, "incorrect instrumentation configuration - please provide container names for all instrumentations"
	} else if instrWithoutContainers == 0 && instrWithContainers == 0 {
		return false, "instrumentation configuration not provided"
	}

	return true, "ok"
}

// Look for duplicates in the provided containers.
func findDuplicatedContainers(ctrs []string) []string {
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

	return duplicates
}

// Set containers for specific instrumentation.
func setInstrumentationLanguageContainers(insts *languageInstrumentations, instrumentationName string, containers string) bool {
	instrs := reflect.ValueOf(insts)
	instLang := instrs.Elem().FieldByName(instrumentationName)

	// Check if the field exists and is a nested struct.
	if !instLang.IsValid() || instLang.Kind() != reflect.Struct {
		return false
	}

	containersField := instLang.FieldByName("Containers")
	// Check if the "Containers" field exists and is assignable.
	if !containersField.IsValid() || !containersField.CanSet() {
		return false
	}
	containersField.SetString(containers)

	return true
}
