// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package podmutation contains the webhook that injects sidecars into pods.
package podmutation

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:rbac:groups="",resources=namespaces;secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch
// +kubebuilder:rbac:groups="apps",resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups="batch",resources=jobs;cronjobs,verbs=get;list;watch

var _ admission.Defaulter[*corev1.Pod] = (*podMutationWebhook)(nil)

// WebhookHandler is a webhook handler that analyzes new pods and injects appropriate sidecars into it.
type WebhookHandler interface {
	admission.Handler
}

// failOpenHandler returns internal error status codes without denying pod admission.
type failOpenHandler struct {
	inner admission.Handler
}

func (h failOpenHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	response := h.inner.Handle(ctx, req)
	if response.Result != nil && response.Result.Code == http.StatusInternalServerError {
		response.Allowed = true
	}
	return response
}

// the implementation.
type podMutationWebhook struct {
	client      client.Client
	logger      logr.Logger
	podMutators []PodMutator
}

// PodMutator mutates a pod.
type PodMutator interface {
	Mutate(ctx context.Context, ns corev1.Namespace, pod corev1.Pod) (corev1.Pod, error)
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(logger logr.Logger, scheme *runtime.Scheme, cl client.Client, podMutators []PodMutator) WebhookHandler {
	defaulter := &podMutationWebhook{
		logger:      logger,
		client:      cl,
		podMutators: podMutators,
	}
	return failOpenHandler{inner: admission.WithDefaulter[*corev1.Pod](scheme, defaulter)}
}

func (p *podMutationWebhook) Default(ctx context.Context, pod *corev1.Pod) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		p.logger.Error(err, "Failed to get admission request from context")
		return apierrors.NewInternalError(err)
	}

	// we use the req.Namespace here because the pod might have not been created yet
	ns := corev1.Namespace{}
	err = p.client.Get(ctx, types.NamespacedName{Name: req.Namespace, Namespace: ""}, &ns)
	if err != nil {
		p.logger.Error(err, "Failed to get namespace", "namespace", req.Namespace)
		return apierrors.NewInternalError(err)
	}

	mutatedPod := *pod.DeepCopy()
	for _, m := range p.podMutators {
		mutatedPod, err = m.Mutate(ctx, ns, mutatedPod)
		if err != nil {
			p.logger.Error(err, "Failed to mutate pod", "namespace", req.Namespace, "name", pod.Name)
			return apierrors.NewInternalError(err)
		}
	}

	*pod = mutatedPod
	return nil
}
