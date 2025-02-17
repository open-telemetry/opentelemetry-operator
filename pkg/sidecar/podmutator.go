// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sidecar

import (
	"context"
	"errors"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
)

var (
	errMultipleInstancesPossible = errors.New("multiple OpenTelemetry Collector instances available, cannot determine which one to select")
	errNoInstancesAvailable      = errors.New("no OpenTelemetry Collector instances available")
	errInstanceNotSidecar        = errors.New("the OpenTelemetry Collector's mode is not set to sidecar")
)

type sidecarPodMutator struct {
	client client.Client
	logger logr.Logger
	config config.Config
}

var _ podmutation.PodMutator = (*sidecarPodMutator)(nil)

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
		return remove(pod), nil
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
		if errors.Is(err, errMultipleInstancesPossible) || errors.Is(err, errNoInstancesAvailable) || errors.Is(err, errInstanceNotSidecar) {
			// we still allow the pod to be created, but we log a message to the operator's logs
			logger.Error(err, "failed to select an OpenTelemetry Collector instance for this pod's sidecar")
			return pod, nil
		}

		// something else happened, better fail here
		return pod, err
	}

	// getting pod references, if any
	references := p.podReferences(ctx, pod.OwnerReferences, ns)
	attributes := getResourceAttributesEnv(ns, references)

	// once it's been determined that a sidecar is desired, none exists yet, and we know which instance it should talk to,
	// we should add the sidecar.
	logger.V(1).Info("injecting sidecar into pod", "otelcol-namespace", otelcol.Namespace, "otelcol-name", otelcol.Name)

	return add(p.config, p.logger, otelcol, pod, attributes)
}

func (p *sidecarPodMutator) getCollectorInstance(ctx context.Context, ns corev1.Namespace, ann string) (v1beta1.OpenTelemetryCollector, error) {
	if strings.EqualFold(ann, "true") {
		return p.selectCollectorInstance(ctx, ns)
	}

	otelcol := v1beta1.OpenTelemetryCollector{}
	var nsnOtelcol types.NamespacedName
	instNamespace, instName, namespaced := strings.Cut(ann, "/")
	if namespaced {
		nsnOtelcol = types.NamespacedName{Name: instName, Namespace: instNamespace}
	} else {
		nsnOtelcol = types.NamespacedName{Name: ann, Namespace: ns.Name}
	}
	err := p.client.Get(ctx, nsnOtelcol, &otelcol)
	if err != nil {
		return otelcol, err
	}

	if otelcol.Spec.Mode != v1beta1.ModeSidecar {
		return v1beta1.OpenTelemetryCollector{}, errInstanceNotSidecar
	}

	return otelcol, nil
}

func (p *sidecarPodMutator) selectCollectorInstance(ctx context.Context, ns corev1.Namespace) (v1beta1.OpenTelemetryCollector, error) {
	var (
		otelcols = v1beta1.OpenTelemetryCollectorList{}
		sidecars []v1beta1.OpenTelemetryCollector
	)

	if err := p.client.List(ctx, &otelcols, client.InNamespace(ns.Name)); err != nil {
		return v1beta1.OpenTelemetryCollector{}, err
	}

	for i := range otelcols.Items {
		coll := otelcols.Items[i]
		if coll.Spec.Mode == v1beta1.ModeSidecar {
			sidecars = append(sidecars, coll)
		}
	}

	switch {
	case len(sidecars) == 0:
		return v1beta1.OpenTelemetryCollector{}, errNoInstancesAvailable
	case len(sidecars) > 1:
		return v1beta1.OpenTelemetryCollector{}, errMultipleInstancesPossible
	default:
		return sidecars[0], nil
	}
}

func (p *sidecarPodMutator) podReferences(ctx context.Context, ownerReferences []metav1.OwnerReference, ns corev1.Namespace) podReferences {
	references := &podReferences{}
	replicaSet := p.getReplicaSetReference(ctx, ownerReferences, ns)
	if replicaSet != nil {
		references.replicaset = replicaSet
		deployment := p.getDeploymentReference(ctx, replicaSet)
		if deployment != nil {
			references.deployment = deployment
		}
	}
	return *references
}

func (p *sidecarPodMutator) getReplicaSetReference(ctx context.Context, ownerReferences []metav1.OwnerReference, ns corev1.Namespace) *appsv1.ReplicaSet {
	replicaSetName := findOwnerReferenceKind(ownerReferences, "ReplicaSet")
	if replicaSetName != "" {
		replicaSet := &appsv1.ReplicaSet{}
		err := p.client.Get(ctx, types.NamespacedName{Name: replicaSetName, Namespace: ns.Name}, replicaSet)
		if err == nil {
			return replicaSet
		}
	}
	return nil
}

func (p *sidecarPodMutator) getDeploymentReference(ctx context.Context, replicaSet *appsv1.ReplicaSet) *appsv1.Deployment {
	deploymentName := findOwnerReferenceKind(replicaSet.OwnerReferences, "Deployment")
	if deploymentName != "" {
		deployment := &appsv1.Deployment{}
		err := p.client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: replicaSet.Namespace}, deployment)
		if err == nil {
			return deployment
		}
	}
	return nil
}

func findOwnerReferenceKind(references []metav1.OwnerReference, kind string) string {
	for _, reference := range references {
		if reference.Kind == kind {
			return reference.Name
		}
	}
	return ""
}
