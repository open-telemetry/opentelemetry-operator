// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
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
	config      config.Config
}

type instrumentationWithContainers struct {
	Instrumentation       *v1alpha1.Instrumentation
	Containers            []string
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

// Check if specific containers are provided for configured instrumentation.
func (langInsts languageInstrumentations) areInstrumentedContainersCorrect() (bool, error) {
	var instrWithoutContainers int
	var instrWithContainers int
	var allContainers []string
	var instrumentationWithNoContainers bool

	// Check for instrumentations with and without containers.
	if langInsts.Java.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Java)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Java)
		allContainers = append(allContainers, langInsts.Java.Containers...)
		if len(langInsts.Java.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.NodeJS.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.NodeJS)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.NodeJS)
		allContainers = append(allContainers, langInsts.NodeJS.Containers...)
		if len(langInsts.NodeJS.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.Python.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Python)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Python)
		allContainers = append(allContainers, langInsts.Python.Containers...)
		if len(langInsts.Python.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.DotNet.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.DotNet)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.DotNet)
		allContainers = append(allContainers, langInsts.DotNet.Containers...)
		if len(langInsts.DotNet.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.ApacheHttpd.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.ApacheHttpd)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.ApacheHttpd)
		allContainers = append(allContainers, langInsts.ApacheHttpd.Containers...)
		if len(langInsts.ApacheHttpd.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.Nginx.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Nginx)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Nginx)
		allContainers = append(allContainers, langInsts.Nginx.Containers...)
		if len(langInsts.Nginx.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.Go.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Go)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Go)
		allContainers = append(allContainers, langInsts.Go.Containers...)
		if len(langInsts.Go.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
	}
	if langInsts.Sdk.Instrumentation != nil {
		instrWithContainers += isInstrWithContainers(langInsts.Sdk)
		instrWithoutContainers += isInstrWithoutContainers(langInsts.Sdk)
		allContainers = append(allContainers, langInsts.Sdk.Containers...)
		if len(langInsts.Sdk.Containers) == 0 {
			instrumentationWithNoContainers = true
		}
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

	enabledInstrumentations := instrWithContainers + instrWithoutContainers

	if enabledInstrumentations > 1 && instrumentationWithNoContainers {
		return false, fmt.Errorf("incorrect instrumentation configuration - please provide container names for all instrumentations")
	}

	return true, nil
}

// Set containers for configured instrumentation.
func (langInsts *languageInstrumentations) setCommonInstrumentedContainers(ns corev1.Namespace, pod corev1.Pod) error {
	containersAnnotation := annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInjectContainerName)
	if err := isValidContainersAnnotation(containersAnnotation); err != nil {
		return err
	}

	var containers []string
	if containersAnnotation == "" {
		return nil
	} else {
		containers = strings.Split(containersAnnotation, ",")
	}

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
	return nil
}

func (langInsts *languageInstrumentations) setLanguageSpecificContainers(ns metav1.ObjectMeta, pod metav1.ObjectMeta) error {
	inst := []struct {
		iwc        *instrumentationWithContainers
		annotation string
	}{
		{
			iwc:        &langInsts.Java,
			annotation: annotationInjectJavaContainersName,
		},
		{
			iwc:        &langInsts.NodeJS,
			annotation: annotationInjectNodeJSContainersName,
		},
		{
			iwc:        &langInsts.Python,
			annotation: annotationInjectPythonContainersName,
		},
		{
			iwc:        &langInsts.DotNet,
			annotation: annotationInjectDotnetContainersName,
		},
		{
			iwc:        &langInsts.Go,
			annotation: annotationInjectGoContainersName,
		},
		{
			iwc:        &langInsts.ApacheHttpd,
			annotation: annotationInjectApacheHttpdContainersName,
		},
		{
			iwc:        &langInsts.Nginx,
			annotation: annotationInjectNginxContainersName,
		},
		{
			iwc:        &langInsts.Sdk,
			annotation: annotationInjectSdkContainersName,
		},
	}

	for _, i := range inst {
		i := i
		if err := setContainersFromAnnotation(i.iwc, i.annotation, ns, pod); err != nil {
			return err
		}
	}
	return nil
}

var _ podmutation.PodMutator = (*instPodMutator)(nil)

func NewMutator(logger logr.Logger, client client.Client, recorder record.EventRecorder, cfg config.Config) *instPodMutator {
	return &instPodMutator{
		Logger: logger,
		Client: client,
		sdkInjector: &sdkInjector{
			logger: logger,
			client: client,
		},
		Recorder: recorder,
		config:   cfg,
	}
}

func (pm *instPodMutator) Mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error) {
	logger := pm.Logger.WithValues("namespace", pod.Namespace)
	if pod.Name != "" {
		logger = logger.WithValues("name", pod.Name)
	} else if pod.GenerateName != "" {
		logger = logger.WithValues("generateName", pod.GenerateName)
	}

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
	if pm.config.EnableJavaAutoInstrumentation() || inst == nil {
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
	if pm.config.EnableNodeJSAutoInstrumentation() || inst == nil {
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
	if pm.config.EnablePythonAutoInstrumentation() || inst == nil {
		insts.Python.Instrumentation = inst
		insts.Python.AdditionalAnnotations = map[string]string{annotationPythonPlatform: annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationPythonPlatform)}
	} else {
		logger.Error(nil, "support for Python auto instrumentation is not enabled")
		pm.Recorder.Event(pod.DeepCopy(), "Warning", "InstrumentationRequestRejected", "support for Python auto instrumentation is not enabled")
	}

	if inst, err = pm.getInstrumentationInstance(ctx, ns, pod, annotationInjectDotNet); err != nil {
		// we still allow the pod to be created, but we log a message to the operator's logs
		logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
		return pod, err
	}
	if pm.config.EnableDotNetAutoInstrumentation() || inst == nil {
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
	if pm.config.EnableGoAutoInstrumentation() || inst == nil {
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
	if pm.config.EnableApacheHttpdAutoInstrumentation() || inst == nil {
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
	if pm.config.EnableNginxAutoInstrumentation() || inst == nil {
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

	err = insts.setCommonInstrumentedContainers(ns, pod)
	if err != nil {
		return pod, err
	}

	if err = pm.validateInstrumentations(ctx, insts, ns.Name); err != nil {
		logger.Error(err, "failed to validate instrumentations")
		return pod, err
	}

	// We retrieve the annotation for podname
	if pm.config.EnableMultiInstrumentation() {
		err = insts.setLanguageSpecificContainers(ns.ObjectMeta, pod.ObjectMeta)
		if err != nil {
			return pod, err
		}

		// We check if provided annotations and instrumentations are valid
		ok, msg := insts.areInstrumentedContainersCorrect()
		if !ok {
			logger.V(1).Error(msg, "skipping instrumentation injection")
			return pod, nil
		}
	}

	// once it's been determined that instrumentation is desired, none exists yet, and we know which instance it should talk to,
	// we should inject the instrumentation.
	modifiedPod := pod
	modifiedPod = pm.sdkInjector.inject(ctx, insts, ns, modifiedPod, pm.config)

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

func (pm *instPodMutator) validateInstrumentations(ctx context.Context, inst languageInstrumentations, podNamespace string) error {
	instrumentations := []struct {
		instrumentation *v1alpha1.Instrumentation
	}{
		{inst.Java.Instrumentation},
		{inst.Python.Instrumentation},
		{inst.NodeJS.Instrumentation},
		{inst.DotNet.Instrumentation},
		{inst.Go.Instrumentation},
		{inst.ApacheHttpd.Instrumentation},
		{inst.Nginx.Instrumentation},
		{inst.Sdk.Instrumentation},
	}
	var errs []error
	for _, i := range instrumentations {
		if i.instrumentation != nil {
			if err := pm.validateInstrumentation(ctx, i.instrumentation, podNamespace); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (pm *instPodMutator) validateInstrumentation(ctx context.Context, inst *v1alpha1.Instrumentation, podNamespace string) error {
	// Check if secret and configmap exists
	// If they don't exist pod cannot start
	var errs []error
	if inst.Spec.Exporter.TLS != nil {
		if inst.Spec.Exporter.TLS.SecretName != "" {
			nsn := types.NamespacedName{Name: inst.Spec.Exporter.TLS.SecretName, Namespace: podNamespace}
			if err := pm.Client.Get(ctx, nsn, &corev1.Secret{}); apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("secret %s with certificates does not exists: %w", nsn.String(), err))
			}
		}
		if inst.Spec.Exporter.TLS.ConfigMapName != "" {
			nsn := types.NamespacedName{Name: inst.Spec.Exporter.TLS.ConfigMapName, Namespace: podNamespace}
			if err := pm.Client.Get(ctx, nsn, &corev1.ConfigMap{}); apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("configmap %s with CA certificate does not exists: %w", nsn.String(), err))
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
