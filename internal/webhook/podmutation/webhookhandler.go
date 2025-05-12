// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package podmutation contains the webhook that injects sidecars into pods.
package podmutation

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:rbac:groups="",resources=namespaces;secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch
// +kubebuilder:rbac:groups="apps",resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch

var _ WebhookHandler = (*podMutationWebhook)(nil)

// WebhookHandler is a webhook handler that analyzes new pods and injects appropriate sidecars into it.
type WebhookHandler interface {
	admission.Handler
}

// the implementation.
type podMutationWebhook struct {
	client      client.Client
	decoder     admission.Decoder
	logger      logr.Logger
	podMutators []PodMutator
	config      config.Config
}

// PodMutator mutates a pod.
type PodMutator interface {
	Mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error)
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(cfg config.Config, logger logr.Logger, decoder admission.Decoder, cl client.Client, podMutators []PodMutator) WebhookHandler {
	return &podMutationWebhook{
		config:      cfg,
		decoder:     decoder,
		logger:      logger,
		client:      cl,
		podMutators: podMutators,
	}
}

func (p *podMutationWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := corev1.Pod{}
	err := p.decoder.Decode(req, &pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// we use the req.Namespace here because the pod might have not been created yet
	ns := corev1.Namespace{}
	err = p.client.Get(ctx, types.NamespacedName{Name: req.Namespace, Namespace: ""}, &ns)
	if err != nil {
		res := admission.Errored(http.StatusInternalServerError, err)
		// By default, admission.Errored sets Allowed to false which blocks pod creation even though the failurePolicy=ignore.
		// Allowed set to true makes sure failure does not block pod creation in case of an error.
		// Using the http.StatusInternalServerError creates a k8s event associated with the replica set.
		// The admission.Allowed("").WithWarnings(err.Error()) or http.StatusBadRequest does not
		// create any event. Additionally, an event/log cannot be created explicitly because the pod name is not known.
		res.Allowed = true
		return res
	}

	for _, m := range p.podMutators {
		pod, err = m.Mutate(ctx, ns, pod)
		if err != nil {
			res := admission.Errored(http.StatusInternalServerError, err)
			res.Allowed = true
			return res
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		res := admission.Errored(http.StatusInternalServerError, err)
		res.Allowed = true
		return res
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
