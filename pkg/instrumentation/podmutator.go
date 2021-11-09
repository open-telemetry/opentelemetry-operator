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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/instrumentation/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhookhandler"
)

var (
	errMultipleInstancesPossible = errors.New("multiple OpenTelemetry Instrumentation instances available, cannot determine which one to select")
	errNoInstancesAvailable      = errors.New("no OpenTelemetry Instrumentation instances available")
	errNoLanguageSpecified       = fmt.Errorf("%s must be set to the desired SDK language", annotationLanguage)
	errUnsupportedLanguage       = errors.New("SDK language not supported, supported languages are (java, nodejs)")
)

type instPodMutator struct {
	Logger logr.Logger
	Client client.Client
}

var _ webhookhandler.PodMutator = (*instPodMutator)(nil)

func NewMutator(logger logr.Logger, client client.Client) *instPodMutator {
	return &instPodMutator{
		Logger: logger,
		Client: client,
	}
}

func (pm *instPodMutator) Mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error) {
	logger := pm.Logger.WithValues("namespace", pod.Namespace, "name", pod.Name)

	// if no annotations are found at all, just return the same pod
	instValue := annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationInject)
	if len(instValue) == 0 {
		logger.V(1).Info("annotation not present in deployment, skipping instrumentation injection")
		return pod, nil
	}

	// is the annotation value 'false'? if so, we need a pod without the instrumentation
	if strings.EqualFold(instValue, "false") {
		logger.V(1).Info("pod explicitly refuses instrumentation injection, attempting to remove instrumentation if it exists")
		return pod, nil
	}

	langValue := annotationValue(ns.ObjectMeta, pod.ObjectMeta, annotationLanguage)

	// which instance should it talk to?
	otelinst, err := pm.getInstrumentationInstance(ctx, ns, instValue, langValue)
	if err != nil {
		if err == errNoInstancesAvailable || err == errMultipleInstancesPossible {
			// we still allow the pod to be created, but we log a message to the operator's logs
			logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod")
			return pod, nil
		}

		// something else happened, better fail here
		return pod, err
	}

	// once it's been determined that instrumentation is desired, none exists yet, and we know which instance it should talk to,
	// we should inject the instrumentation.
	logger.V(1).Info("injecting instrumentation into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
	return inject(pm.Logger, otelinst, pod, langValue), nil
}

func (pm *instPodMutator) getInstrumentationInstance(ctx context.Context, ns corev1.Namespace, instValue string, langValue string) (v1alpha1.Instrumentation, error) {
	otelInst := v1alpha1.Instrumentation{}

	if len(langValue) == 0 {
		return otelInst, errNoLanguageSpecified
	}
	if langValue != "java" && langValue != "nodejs" {
		return otelInst, errUnsupportedLanguage
	}

	if strings.EqualFold(instValue, "true") {
		return pm.selectInstrumentationInstanceFromNamespace(ctx, ns)
	}

	err := pm.Client.Get(ctx, types.NamespacedName{Name: instValue, Namespace: ns.Name}, &otelInst)
	if err != nil {
		return otelInst, err
	}

	return otelInst, nil
}

func (pm *instPodMutator) selectInstrumentationInstanceFromNamespace(ctx context.Context, ns corev1.Namespace) (v1alpha1.Instrumentation, error) {
	var otelInsts v1alpha1.InstrumentationList
	if err := pm.Client.List(ctx, &otelInsts, client.InNamespace(ns.Name)); err != nil {
		return v1alpha1.Instrumentation{}, err
	}

	switch s := len(otelInsts.Items); {
	case s == 0:
		return v1alpha1.Instrumentation{}, errNoInstancesAvailable
	case s > 1:
		return v1alpha1.Instrumentation{}, errMultipleInstancesPossible
	default:
		return otelInsts.Items[0], nil
	}
}
