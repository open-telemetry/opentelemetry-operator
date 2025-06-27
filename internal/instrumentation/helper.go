// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

var defaultSize = resource.MustParse("200Mi")

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
		if slices.Contains([]string{
			dotnetInitContainerName,
			javaInitContainerName,
			nodejsInitContainerName,
			pythonInitContainerName,
			apacheAgentInitContainerName,
			apacheAgentCloneContainerName,
		}, cont.Name) {
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
	countMap := make(map[string]int)
	var duplicates []string
	for _, str := range ctrs {
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
	if len(inst.Containers) > 0 {
		return 1
	}

	return 0
}

// Return positive for instrumentation without defined containers.
func isInstrWithoutContainers(inst instrumentationWithContainers) int {
	if len(inst.Containers) == 0 {
		return 1
	}

	return 0
}

// Return volume if defined, otherwise return emptyDir with given name and size limit.
func instrVolume(volumeClaimTemplate corev1.PersistentVolumeClaimTemplate, name string, quantity *resource.Quantity) corev1.Volume {
	if !reflect.ValueOf(volumeClaimTemplate).IsZero() {
		return corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Ephemeral: &corev1.EphemeralVolumeSource{
					VolumeClaimTemplate: &volumeClaimTemplate,
				},
			},
		}
	}

	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: volumeSize(quantity),
			},
		}}
}

func volumeSize(quantity *resource.Quantity) *resource.Quantity {
	if quantity == nil {
		return &defaultSize
	}
	return quantity
}

func isValidContainersAnnotation(containersAnnotation string) error {
	if containersAnnotation == "" {
		return nil
	}

	matched, err := regexp.MatchString("^[a-zA-Z0-9-,]+$", containersAnnotation)
	if err != nil {
		return fmt.Errorf("error while checking for instrumentation container annotations %w", err)
	}
	if !matched {
		return fmt.Errorf("not valid characters included in the instrumentation container annotation %s", containersAnnotation)
	}
	return nil
}

// setContainersFromAnnotation sets the containers associated to one intrumentation based on the content of the provided annotation.
func setContainersFromAnnotation(inst *instrumentationWithContainers, annotation string, ns metav1.ObjectMeta, pod metav1.ObjectMeta) error {
	annotationValue := annotationValue(ns, pod, annotation)
	if annotationValue == "" {
		return nil
	}

	if err := isValidContainersAnnotation(annotationValue); err != nil {
		return err
	}
	languageContainers := strings.Split(annotationValue, ",")
	inst.Containers = append(inst.Containers, languageContainers...)
	return nil
}
