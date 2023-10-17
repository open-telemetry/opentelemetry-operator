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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

var (
	errMultipleInstancesPossible = errors.New("multiple OpenTelemetry Instrumentation instances available, cannot determine which one to select")
	errNoInstancesAvailable      = errors.New("no OpenTelemetry Instrumentation instances available")
)

type instPodMutator struct {
	Client      client.Client
	sdkInjector *sdkInjector
	Logger      logr.Logger
	Recorder    record.EventRecorder
}

type instrumentationWithContainers struct {
	Instrumentation       *v1alpha1.Instrumentation
	Containers            string
	AdditionalAnnotations map[string]string
}

type languageInstrumentations struct {
	Java        instrumentationWithContainers
	NodeJS      instrumentationWithContainers
	Python      instrumentationWithContainers
	DotNet      instrumentationWithContainers
	ApacheHttpd instrumentationWithContainers
	Nginx       instrumentationWithContainers
	Go          instrumentationWithContainers
	Sdk         instrumentationWithContainers
}

// Check if single instrumentation is configured for Pod and return which is configured.
func (langInsts languageInstrumentations) isSingleInstrumentationEnabled() bool {
	count := 0

	if langInsts.Java.Instrumentation != nil {
		count++
	}
	if langInsts.NodeJS.Instrumentation != nil {
		count++
	}
	if langInsts.Python.Instrumentation != nil {
		count++
	}
	if langInsts.DotNet.Instrumentation != nil {
		count++
	}
	if langInsts.ApacheHttpd.Instrumentation != nil {
		count++
	}
	if langInsts.Nginx.Instrumentation != nil {
		count++
	}
	if langInsts.Go.Instrumentation != nil {
		count++
	}
	if langInsts.Sdk.Instrumentation != nil {
		count++
	}

	return count == 1
}

// Check if specific containers are provided for configured instrumentation.
func (langInsts languageInstrumentations) areContainerNamesConfiguredForMultipleInstrumentations() (bool, error) {
	var instrWithoutContainers int
	var instrWithContainers int
	var allContainers []string

	// Check for instrumentations with and without containers.
	if langInsts.Java.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Java)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Java)
		allContainers = append(allContainers, langInsts.Java.Containers)
	}
	if langInsts.NodeJS.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.NodeJS)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.NodeJS)
		allContainers = append(allContainers, langInsts.NodeJS.Containers)
	}
	if langInsts.Python.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Python)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Python)
		allContainers = append(allContainers, langInsts.Python.Containers)
	}
	if langInsts.DotNet.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.DotNet)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.DotNet)
		allContainers = append(allContainers, langInsts.DotNet.Containers)
	}
	if langInsts.ApacheHttpd.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.ApacheHttpd)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.ApacheHttpd)
		allContainers = append(allContainers, langInsts.ApacheHttpd.Containers)
	}
	if langInsts.Nginx.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Nginx)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Nginx)
		allContainers = append(allContainers, langInsts.Nginx.Containers)
	}
	if langInsts.Go.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Go)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Go)
		allContainers = append(allContainers, langInsts.Go.Containers)
	}
	if langInsts.Sdk.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Sdk)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Sdk)
		allContainers = append(allContainers, langInsts.Sdk.Containers)
	}

	// Look for duplicated containers.
	containerDuplicates := findDuplicatedContainers(allContainers)
	if containerDuplicates != nil {
		return false, containerDuplicates
	}

	// Look for mixed multiple instrumentations with and without container names.
	if instrWithoutContainers > 0 && instrWithContainers > 0 {
		return false, fmt.Errorf("incorrect instrumentation configuration - please provide container names for all instrumentations")
	}

	// Look for multiple instrumentations without container names.
	if instrWithoutContainers > 1 && instrWithContainers == 0 {
		return false, fmt.Errorf("incorrect instrumentation configuration - please provide container names for all instrumentations")
	}

	if instrWithoutContainers == 0 && instrWithContainers == 0 {
		return false, fmt.Errorf("instrumentation configuration not provided")
	}

	return true, nil
}

// Set containers for configured instrumentation.
func (langInsts *languageInstrumentations) setInstrumentationLanguageContainers(containers string) {
	if langInsts.Java.Instrumentation != nil {
		langInsts.Java.Containers = containers
	}
	if langInsts.NodeJS.Instrumentation != nil {
		langInsts.NodeJS.Containers = containers
	}
	if langInsts.Python.Instrumentation != nil {
		langInsts.Python.Containers = containers
	}
	if langInsts.DotNet.Instrumentation != nil {
		langInsts.DotNet.Containers = containers
	}
	if langInsts.ApacheHttpd.Instrumentation != nil {
		langInsts.ApacheHttpd.Containers = containers
	}
	if langInsts.Nginx.Instrumentation != nil {
		langInsts.Nginx.Containers = containers
	}
	if langInsts.Go.Instrumentation != nil {
		langInsts.Go.Containers = containers
	}
	if langInsts.Sdk.Instrumentation != nil {
		langInsts.Sdk.Containers = containers
	}
}

var _ podmutation.PodMutator = (*instPodMutator)(nil)

func NewMutator(logger logr.Logger, client client.Client, recorder record.EventRecorder) *instPodMutator {
	return &instPodMutator{
		Logger: logger,
		Client: client,
		sdkInjector: &sdkInjector{
			logger: logger,
			client: client,
		},
		Recorder: recorder,
	}
}

func (pm *instPodMutator) Mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error) {
	logger := pm.Logger.WithValues("namespace", pod.Namespace, "name", pod.Name)

	// We check if Pod is already instrumented.
	if isAutoInstrumentationInjected(pod) {
		logger.Info("Skipping pod instrumentation - already instrumented")
		return pod, nil
	}

	var inst *v1alpha1.Instrumentation
	var err error

	insts := languageInstrumentations{}

	// We bail out if any annotation fails to process.

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectJava); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnableJavaAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.Java.Instrumentation = inst
	} else {
		logger.Error(nil, "support for Java auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for Java auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectNodeJS); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnableNodeJSAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.NodeJS.Instrumentation = inst
	} else {
		logger.Error(nil, "support for NodeJS auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for NodeJS auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectPython); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnablePythonAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.Python.Instrumentation = inst
	} else {
		logger.Error(nil, "support for Python auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for Python auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectDotNet); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnableDotnetAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.DotNet.Instrumentation = inst
		insts.DotNet.AdditionalAnnotations = map[string]string{annotationDotNetRuntime: annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationDotNetRuntime)}
	} else {
		logger.Error(nil, "support for .NET auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for .NET auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectGo); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnableGoAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.Go.Instrumentation = inst
	} else {
		logger.Error(err, "support for Go auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for Go auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectApacheHttpd); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnableApacheHTTPAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.ApacheHttpd.Instrumentation = inst
	} else {
		logger.Error(nil, "support for Apache HTTPD auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for Apache HTTPD auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectNginx); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if featuregate.EnableNginxAutoInstrumentationSupport.IsEnabled() || inst == nil {
		insts.Nginx.Instrumentation = inst
	} else {
		logger.Error(nil, "support for Nginx auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for Nginx auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectSdk); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	insts.Sdk.Instrumentation = inst

	if insts.Java.Instrumentation == nil && insts.NodeJS.Instrumentation == nil && insts.Python.Instrumentation == nil &&
		insts.DotNet.Instrumentation == nil && insts.Go.Instrumentation == nil && insts.ApacheHttpd.Instrumentation == nil &&
		insts.Nginx.Instrumentation == nil &&
		insts.Sdk.Instrumentation == nil {

		logger.V(1).Info("annotation not present in deployment, skipping instrumentation injection")
		return pod, nil
	}

	// We retrieve the annotation for podname
	if featuregate.EnableMultiInstrumentationSupport.IsEnabled() {
		// We use annotations specific for instrumentation language
		insts.Java.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectJavaContainersName)
		insts.NodeJS.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectNodeJSContainersName)
		insts.Python.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectPythonContainersName)
		insts.DotNet.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectDotnetContainersName)
		insts.Go.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectGoContainersName)
		insts.ApacheHttpd.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectApacheHttpdContainersName)
		insts.Nginx.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectNginxContainersName)
		insts.Sdk.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectSdkContainersName)

		// We check if provided annotations and instrumentations are valid
		ok, msg := insts.areContainerNamesConfiguredForMultipleInstrumentations()
		if !ok {
			logger.V(1).Error(msg, "skipping instrumentation injection")
			return pod, nil
		}
	} else {
		// We use general annotation for container names
		// only when multi instrumentation is disabled
		singleInstrEnabled := insts.isSingleInstrumentationEnabled()
		if singleInstrEnabled {
			generalContainerNames := annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectContainerName)
			insts.setInstrumentationLanguageContainers(generalContainerNames)
		} else {
			logger.V(1).Error(fmt.Errorf("multiple injection annotations present"), "skipping instrumentation injection")
			return pod, nil
		}

	}

	// once it's been determined that instrumentation is desired, none exists yet, and we know which instance it should talk to,
	// we should inject the instrumentation.
	modifiedPod := pod
	modifiedPod = pm.sdkInjector.inject(ctx, insts, ns, modifiedPod)

	return modifiedPod, nil
}

func (pm *instPodMutator) getInstrumentationInstance(ctx context.Context, ns corev1.Namespace, pod corev1.Pod, instAnnotation string) (*v1alpha1.Instrumentation, error) {
	instValue := annotationValue(ns.ObjectMeta, pod.ObjectMeta, instAnnotation)

	if len(instValue) == 0 || strings.EqualFold(instValue, "false") {
		return nil, nil
	}

	if strings.EqualFold(instValue, "true") {
		return pm.selectInstrumentationInstanceFromNamespace(ctx, ns)
	}

	var instNamespacedName types.NamespacedName
	if instNamespace, instName, namespaced := strings.Cut(instValue, "/"); namespaced {
		instNamespacedName = types.NamespacedName{Name: instName, Namespace: instNamespace}
	} else {
		instNamespacedName = types.NamespacedName{Name: instValue, Namespace: ns.Name}
	}

	otelInst := &v1alpha1.Instrumentation{}
	err := pm.Client.Get(ctx, instNamespacedName, otelInst)
	if err != nil {
		return nil, err
	}

	return otelInst, nil
}

func (pm *instPodMutator) selectInstrumentationInstanceFromNamespace(ctx context.Context, ns corev1.Namespace) (*v1alpha1.Instrumentation, error) {
	var otelInsts v1alpha1.InstrumentationList
	if err := pm.Client.List(ctx, &otelInsts, client.InNamespace(ns.Name)); err != nil {
		return nil, err
	}

	switch s := len(otelInsts.Items); {
	case s == 0:
		return nil, errNoInstancesAvailable
	case s > 1:
		return nil, errMultipleInstancesPossible
	default:
		return &otelInsts.Items[0], nil
	}
}
