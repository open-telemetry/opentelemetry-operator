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

// Package podinjector contains the webhook that injects sidecars into pods.
package podinjector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/config"
	"github.com/signalfx/splunk-otel-operator/pkg/sidecar"
)

var (
	ErrMultipleInstancesPossible = errors.New("multiple OpenTelemetry Collector instances available, cannot determine which one to select")
	ErrNoInstancesAvailable      = errors.New("no OpenTelemetry Collector instances available")
	ErrInstanceNotSidecar        = errors.New("the OpenTelemetry Collector's mode is not set to sidecar")
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io,sideEffects=none,admissionReviewVersions=v1;v1beta1
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;watch
// +kubebuilder:rbac:groups=splunk.com,resources=splunkotelagents,verbs=get;list;watch

var _ PodSidecarInjector = (*podSidecarInjector)(nil)

// PodSidecarInjector is a webhook handler that analyzes new pods and injects appropriate sidecars into it.
type PodSidecarInjector interface {
	admission.Handler
	admission.DecoderInjector
}

// the implementation.
type podSidecarInjector struct {
	config  config.Config
	logger  logr.Logger
	client  client.Client
	decoder *admission.Decoder
}

// NewPodSidecarInjector creates a new PodSidecarInjector.
func NewPodSidecarInjector(cfg config.Config, logger logr.Logger, cl client.Client) PodSidecarInjector {
	return &podSidecarInjector{
		config: cfg,
		logger: logger,
		client: cl,
	}
}

func (p *podSidecarInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := corev1.Pod{}
	err := p.decoder.Decode(req, &pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// we use the req.Namespace here because the pod might have not been created yet
	ns := corev1.Namespace{}
	err = p.client.Get(ctx, types.NamespacedName{Name: req.Namespace, Namespace: ""}, &ns)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	pod, err = p.mutate(ctx, ns, pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (p *podSidecarInjector) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}

func (p *podSidecarInjector) mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error) {
	logger := p.logger.WithValues("namespace", pod.Namespace, "name", pod.Name)

	// if no annotations are found at all, just return the same pod
	annValue := sidecar.AnnotationValue(ns, pod)
	if len(annValue) == 0 {
		logger.V(1).Info("annotation not present in deployment, skipping sidecar injection")
		return pod, nil
	}

	// is the annotation value 'false'? if so, we need a pod without the sidecar (ie, remove if exists)
	if strings.EqualFold(annValue, "false") {
		logger.V(1).Info("pod explicitly refuses sidecar injection, attempting to remove sidecar if it exists")
		return sidecar.Remove(pod)
	}

	// from this point and on, a sidecar is wanted

	// check whether there's a sidecar already -- return the same pod if that's the case.
	if sidecar.ExistsIn(pod) {
		logger.V(1).Info("pod already has sidecar in it, skipping injection")
		return pod, nil
	}

	// which instance should it talk to?
	otelcol, err := p.getCollectorInstance(ctx, ns, annValue)
	if err != nil {
		if err == ErrMultipleInstancesPossible || err == ErrNoInstancesAvailable || err == ErrInstanceNotSidecar {
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
	return sidecar.Add(p.config, p.logger, otelcol, pod)
}

func (p *podSidecarInjector) getCollectorInstance(ctx context.Context, ns corev1.Namespace, ann string) (v1alpha1.SplunkOtelAgent, error) {
	if strings.EqualFold(ann, "true") {
		return p.selectCollectorInstance(ctx, ns)
	}

	otelcol := v1alpha1.SplunkOtelAgent{}
	err := p.client.Get(ctx, types.NamespacedName{Name: ann, Namespace: ns.Name}, &otelcol)
	if err != nil {
		return otelcol, err
	}

	if otelcol.Spec.Mode != v1alpha1.ModeSidecar {
		return v1alpha1.SplunkOtelAgent{}, ErrInstanceNotSidecar
	}

	return otelcol, nil
}

func (p *podSidecarInjector) selectCollectorInstance(ctx context.Context, ns corev1.Namespace) (v1alpha1.SplunkOtelAgent, error) {
	var (
		otelcols = v1alpha1.SplunkOtelAgentList{}
		sidecars []v1alpha1.SplunkOtelAgent
	)

	if err := p.client.List(ctx, &otelcols, client.InNamespace(ns.Name)); err != nil {
		return v1alpha1.SplunkOtelAgent{}, err
	}

	for i := range otelcols.Items {
		coll := otelcols.Items[i]
		if coll.Spec.Mode == v1alpha1.ModeSidecar {
			sidecars = append(sidecars, coll)
		}
	}

	switch {
	case len(sidecars) == 0:
		return v1alpha1.SplunkOtelAgent{}, ErrNoInstancesAvailable
	case len(sidecars) > 1:
		return v1alpha1.SplunkOtelAgent{}, ErrMultipleInstancesPossible
	default:
		return sidecars[0], nil
	}
}
