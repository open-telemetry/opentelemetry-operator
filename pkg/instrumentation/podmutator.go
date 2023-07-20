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
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhookhandler"
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
	Instrumentation *v1alpha1.Instrumentation
	Containers      string
}

type languageInstrumentations struct {
	Java        instrumentationWithContainers
	NodeJS      instrumentationWithContainers
	Python      instrumentationWithContainers
	DotNet      instrumentationWithContainers
	ApacheHttpd instrumentationWithContainers
	Go          instrumentationWithContainers
	Sdk         instrumentationWithContainers
}

var _ webhookhandler.PodMutator = (*instPodMutator)(nil)

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

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectSdk); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	insts.Sdk.Instrumentation = inst

	if insts.Java.Instrumentation == nil && insts.NodeJS.Instrumentation == nil && insts.Python.Instrumentation == nil &&
		insts.DotNet.Instrumentation == nil && insts.Go.Instrumentation == nil && insts.ApacheHttpd.Instrumentation == nil &&
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
		insts.Sdk.Containers = annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectSdkContainersName)

		// We check if provided annotations and instrumentations are valid
		ok, msg := areContainerNamesConfiguredForMultipleInstrumentations(insts)
		if !ok {
			logger.V(1).Info("skipping instrumentation injection", "reason", msg)
			return pod, nil
		}

	} else {
		// We use general annotation for container names
		// only when multi instrumentation is disabled
		singleInstrEnabled, instrName := isSingleInstrumentationEnabled(insts)
		if singleInstrEnabled {
			generalContainerNames := annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectContainerName)
			containersSet := setInstrumentationLanguageContainers(&insts, instrName, generalContainerNames)
			if !containersSet {
				logger.V(1).Info("skipping instrumentation injection, unknown instrumentation name")
				return pod, nil
			}
		} else {
			logger.V(1).Info("multiple injection annotations present, skipping instrumentation injection")
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
