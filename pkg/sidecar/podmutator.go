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

package sidecar

import (
	"context"
	"errors"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/collector/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhookhandler"
)

var (
	errMultipleInstancesPossible = errors.New("multiple OpenTelemetry Collector instances available, cannot determine which one to select")
	errNoInstancesAvailable      = errors.New("no OpenTelemetry Collector instances available")
	errInstanceNotSidecar        = errors.New("the OpenTelemetry Collector's mode is not set to sidecar")
)

type sidecarPodMutator struct {
	config config.Config
	logger logr.Logger
	client client.Client
}

var _ webhookhandler.PodMutator = (*sidecarPodMutator)(nil)

func NewMutator(logger logr.Logger, config config.Config, client client.Client) *sidecarPodMutator {
	return &sidecarPodMutator{
		config: config,
		logger: logger,
		client: client,
	}
}

func (p *sidecarPodMutator) Mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error) {
	logger := p.logger.WithValues("namespace", pod.Namespace, "name", pod.Name)

	// if no annotations are found at all, just return the same pod
	annValue := annotationValue(ns, pod)
	if len(annValue) == 0 {
		logger.V(1).Info("annotation not present in deployment, skipping sidecar injection")
		return pod, nil
	}

	// is the annotation value 'false'? if so, we need a pod without the sidecar (ie, remove if exists)
	if strings.EqualFold(annValue, "false") {
		logger.V(1).Info("pod explicitly refuses sidecar injection, attempting to remove sidecar if it exists")
		return remove(pod)
	}

	// from this point and on, a sidecar is wanted

	// check whether there's a sidecar already -- return the same pod if that's the case.
	if existsIn(pod) {
		logger.V(1).Info("pod already has sidecar in it, skipping injection")
		return pod, nil
	}

	// which instance should it talk to?
	otelcol, err := p.getCollectorInstance(ctx, ns, annValue)
	if err != nil {
		if err == errMultipleInstancesPossible || err == errNoInstancesAvailable || err == errInstanceNotSidecar {
			// we still allow the pod to be created, but we log a message to the operator's logs
			logger.Error(err, "failed to select an OpenTelemetry Collector instance for this pod's sidecar")
			return pod, nil
		}

		// something else happened, better fail here
		return pod, err
	}

	// once it's been determined that a sidecar is desired, none exists yet, and we know which instance it should talk to,
	// we should add the sidecar.
	logger.V(1).Info("injecting sidecar into pod", "otelcol-namespace", otelcol.Namespace, "otelcol-name", otelcol.Name)
	return add(p.config, p.logger, otelcol, pod)
}

func (p *sidecarPodMutator) getCollectorInstance(ctx context.Context, ns corev1.Namespace, ann string) (v1alpha1.OpenTelemetryCollector, error) {
	if strings.EqualFold(ann, "true") {
		return p.selectCollectorInstance(ctx, ns)
	}

	otelcol := v1alpha1.OpenTelemetryCollector{}
	err := p.client.Get(ctx, types.NamespacedName{Name: ann, Namespace: ns.Name}, &otelcol)
	if err != nil {
		return otelcol, err
	}

	if otelcol.Spec.Mode != v1alpha1.ModeSidecar {
		return v1alpha1.OpenTelemetryCollector{}, errInstanceNotSidecar
	}

	return otelcol, nil
}

func (p *sidecarPodMutator) selectCollectorInstance(ctx context.Context, ns corev1.Namespace) (v1alpha1.OpenTelemetryCollector, error) {
	var (
		otelcols = v1alpha1.OpenTelemetryCollectorList{}
		sidecars []v1alpha1.OpenTelemetryCollector
	)

	if err := p.client.List(ctx, &otelcols, client.InNamespace(ns.Name)); err != nil {
		return v1alpha1.OpenTelemetryCollector{}, err
	}

	for i := range otelcols.Items {
		coll := otelcols.Items[i]
		if coll.Spec.Mode == v1alpha1.ModeSidecar {
			sidecars = append(sidecars, coll)
		}
	}

	switch {
	case len(sidecars) == 0:
		return v1alpha1.OpenTelemetryCollector{}, errNoInstancesAvailable
	case len(sidecars) > 1:
		return v1alpha1.OpenTelemetryCollector{}, errMultipleInstancesPossible
	default:
		return sidecars[0], nil
	}
}
