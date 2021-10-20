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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/instrumentation/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhookhandler"
)

var (
	errMultipleInstancesPossible = errors.New("multiple OpenTelemetry Instrumentation instances available, cannot determine which one to select")
	errNoInstancesAvailable      = errors.New("no OpenTelemetry Instrumentation instances available")
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
	annValue := annotationValue(ns.ObjectMeta, pod.ObjectMeta)
	if len(annValue) == 0 {
		logger.V(1).Info("annotation not present in deployment, skipping sidecar injection")
		return pod, nil
	}

	// is the annotation value 'false'? if so, we need a pod without the sidecar
	if strings.EqualFold(annValue, "false") {
		logger.V(1).Info("pod explicitly refuses sidecar injection, attempting to remove sidecar if it exists")
		return pod, nil
	}

	// which instance should it talk to?
	otelinst, err := pm.getInstrumentationInstance(ctx, ns, annValue)
	if err != nil {
		if err == errNoInstancesAvailable || err == errMultipleInstancesPossible {
			// we still allow the pod to be created, but we log a message to the operator's logs
			logger.Error(err, "failed to select an OpenTelemetry Instrumentation instance for this pod's sidecar")
			return pod, nil
		}

		// something else happened, better fail here
		return pod, err
	}

	// once it's been determined that a sidecar is desired, none exists yet, and we know which instance it should talk to,
	// we should inject the sidecar.
	logger.V(1).Info("injecting sidecar into pod", "otelinst-namespace", otelinst.Namespace, "otelinst-name", otelinst.Name)
	return inject(pm.Logger, otelinst, pod), nil
}

func (pm *instPodMutator) getInstrumentationInstance(ctx context.Context, ns corev1.Namespace, ann string) (v1alpha1.Instrumentation, error) {
	if strings.EqualFold(ann, "true") {
		return pm.selectInstrumentationInstanceFromNamespace(ctx, ns)
	}

	otelInst := v1alpha1.Instrumentation{}
	err := pm.Client.Get(ctx, types.NamespacedName{Name: ann, Namespace: ns.Name}, &otelInst)
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
