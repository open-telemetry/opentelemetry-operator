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

package manifests

import (
	"fmt"
	"reflect"

	"dario.cat/mergo"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyV1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

type ImmutableFieldChangeErr struct {
	Field string
}

func (e *ImmutableFieldChangeErr) Error() string {
	return fmt.Sprintf("Immutable field change attempted: %s", e.Field)
}

var (
	ImmutableChangeErr *ImmutableFieldChangeErr
)

// MutateFuncFor returns a mutate function based on the
// existing resource's concrete type. It supports currently
// only the following types or else panics:
// - ConfigMap
// - Service
// - ServiceAccount
// - ClusterRole
// - ClusterRoleBinding
// - Role
// - RoleBinding
// - Deployment
// - DaemonSet
// - StatefulSet
// - ServiceMonitor
// - Ingress
// - HorizontalPodAutoscaler
// - Route
// - Secret
// - TargetAllocator
// In order for the operator to reconcile other types, they must be added here.
// The function returned takes no arguments but instead uses the existing and desired inputs here. Existing is expected
// to be set by the controller-runtime package through a client get call.
func MutateFuncFor(existing, desired client.Object) controllerutil.MutateFn {
	return func() error {
		// Get the existing annotations and override any conflicts with the desired annotations
		// This will preserve any annotations on the existing set.
		existingAnnotations := existing.GetAnnotations()
		if err := mergeWithOverride(&existingAnnotations, desired.GetAnnotations()); err != nil {
			return err
		}
		existing.SetAnnotations(existingAnnotations)

		// Get the existing labels and override any conflicts with the desired labels
		// This will preserve any labels on the existing set.
		existingLabels := existing.GetLabels()
		if err := mergeWithOverride(&existingLabels, desired.GetLabels()); err != nil {
			return err
		}
		existing.SetLabels(existingLabels)

		if ownerRefs := desired.GetOwnerReferences(); len(ownerRefs) > 0 {
			existing.SetOwnerReferences(ownerRefs)
		}

		switch existing.(type) {
		case *corev1.ConfigMap:
			cm := existing.(*corev1.ConfigMap)
			wantCm := desired.(*corev1.ConfigMap)
			mutateConfigMap(cm, wantCm)

		case *corev1.Service:
			svc := existing.(*corev1.Service)
			wantSvc := desired.(*corev1.Service)
			mutateService(svc, wantSvc)

		case *corev1.ServiceAccount:
			sa := existing.(*corev1.ServiceAccount)
			wantSa := desired.(*corev1.ServiceAccount)
			mutateServiceAccount(sa, wantSa)

		case *rbacv1.ClusterRole:
			cr := existing.(*rbacv1.ClusterRole)
			wantCr := desired.(*rbacv1.ClusterRole)
			mutateClusterRole(cr, wantCr)

		case *rbacv1.ClusterRoleBinding:
			crb := existing.(*rbacv1.ClusterRoleBinding)
			wantCrb := desired.(*rbacv1.ClusterRoleBinding)
			mutateClusterRoleBinding(crb, wantCrb)

		case *rbacv1.Role:
			r := existing.(*rbacv1.Role)
			wantR := desired.(*rbacv1.Role)
			mutateRole(r, wantR)

		case *rbacv1.RoleBinding:
			rb := existing.(*rbacv1.RoleBinding)
			wantRb := desired.(*rbacv1.RoleBinding)
			mutateRoleBinding(rb, wantRb)

		case *appsv1.Deployment:
			dpl := existing.(*appsv1.Deployment)
			wantDpl := desired.(*appsv1.Deployment)
			return mutateDeployment(dpl, wantDpl)

		case *appsv1.DaemonSet:
			dpl := existing.(*appsv1.DaemonSet)
			wantDpl := desired.(*appsv1.DaemonSet)
			return mutateDaemonset(dpl, wantDpl)

		case *appsv1.StatefulSet:
			sts := existing.(*appsv1.StatefulSet)
			wantSts := desired.(*appsv1.StatefulSet)
			return mutateStatefulSet(sts, wantSts)

		case *monitoringv1.ServiceMonitor:
			svcMonitor := existing.(*monitoringv1.ServiceMonitor)
			wantSvcMonitor := desired.(*monitoringv1.ServiceMonitor)
			mutateServiceMonitor(svcMonitor, wantSvcMonitor)

		case *monitoringv1.PodMonitor:
			podMonitor := existing.(*monitoringv1.PodMonitor)
			wantPodMonitor := desired.(*monitoringv1.PodMonitor)
			mutatePodMonitor(podMonitor, wantPodMonitor)

		case *networkingv1.Ingress:
			ing := existing.(*networkingv1.Ingress)
			wantIng := desired.(*networkingv1.Ingress)
			mutateIngress(ing, wantIng)

		case *autoscalingv2.HorizontalPodAutoscaler:
			existingHPA := existing.(*autoscalingv2.HorizontalPodAutoscaler)
			desiredHPA := desired.(*autoscalingv2.HorizontalPodAutoscaler)
			mutateAutoscalingHPA(existingHPA, desiredHPA)

		case *policyV1.PodDisruptionBudget:
			existingPDB := existing.(*policyV1.PodDisruptionBudget)
			desiredPDB := desired.(*policyV1.PodDisruptionBudget)
			mutatePolicyV1PDB(existingPDB, desiredPDB)

		case *routev1.Route:
			rt := existing.(*routev1.Route)
			wantRt := desired.(*routev1.Route)
			mutateRoute(rt, wantRt)

		case *corev1.Secret:
			pr := existing.(*corev1.Secret)
			wantPr := desired.(*corev1.Secret)
			mutateSecret(pr, wantPr)

		case *cmv1.Certificate:
			cert := existing.(*cmv1.Certificate)
			wantCert := desired.(*cmv1.Certificate)
			mutateCertificate(cert, wantCert)

		case *cmv1.Issuer:
			issuer := existing.(*cmv1.Issuer)
			wantIssuer := desired.(*cmv1.Issuer)
			mutateIssuer(issuer, wantIssuer)

		case *v1alpha1.TargetAllocator:
			ta := existing.(*v1alpha1.TargetAllocator)
			wantTa := desired.(*v1alpha1.TargetAllocator)
			mutateTargetAllocator(ta, wantTa)

		default:
			t := reflect.TypeOf(existing).String()
			return fmt.Errorf("missing mutate implementation for resource type: %s", t)
		}
		return nil
	}
}

func mergeWithOverride(dst, src interface{}) error {
	return mergo.Merge(dst, src, mergo.WithOverride)
}

func mutateSecret(existing, desired *corev1.Secret) {
	existing.Labels = desired.Labels
	existing.Annotations = desired.Annotations
	existing.Data = desired.Data
}

func mutateConfigMap(existing, desired *corev1.ConfigMap) {
	existing.BinaryData = desired.BinaryData
	existing.Data = desired.Data
}

func mutateServiceAccount(existing, desired *corev1.ServiceAccount) {
	existing.Labels = desired.Labels
}

func mutateClusterRole(existing, desired *rbacv1.ClusterRole) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Rules = desired.Rules
}

func mutateClusterRoleBinding(existing, desired *rbacv1.ClusterRoleBinding) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Subjects = desired.Subjects
}

func mutateRole(existing, desired *rbacv1.Role) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Rules = desired.Rules
	// This role can exists in a different namespace than the otel collector, so we need to remove the owner references
	// since cross namespace owner references are not allowed
	existing.SetOwnerReferences(nil)
}

func mutateRoleBinding(existing, desired *rbacv1.RoleBinding) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Subjects = desired.Subjects
	// This role binding can exists in a different namespace than the otel collector, so we need to remove the owner references
	// since cross namespace owner references are not allowed
	existing.SetOwnerReferences(nil)
}

func mutateAutoscalingHPA(existing, desired *autoscalingv2.HorizontalPodAutoscaler) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutatePolicyV1PDB(existing, desired *policyV1.PodDisruptionBudget) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateIngress(existing, desired *networkingv1.Ingress) {
	existing.Labels = desired.Labels
	existing.Annotations = desired.Annotations
	existing.Spec.DefaultBackend = desired.Spec.DefaultBackend
	existing.Spec.Rules = desired.Spec.Rules
	existing.Spec.TLS = desired.Spec.TLS
}

func mutateRoute(existing, desired *routev1.Route) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateServiceMonitor(existing, desired *monitoringv1.ServiceMonitor) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutatePodMonitor(existing, desired *monitoringv1.PodMonitor) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateTargetAllocator(existing, desired *v1alpha1.TargetAllocator) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateService(existing, desired *corev1.Service) {
	existing.Spec.Ports = desired.Spec.Ports
	existing.Spec.Selector = desired.Spec.Selector
}

func mutateDaemonset(existing, desired *appsv1.DaemonSet) error {
	if !existing.CreationTimestamp.IsZero() {
		if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
			return &ImmutableFieldChangeErr{Field: "Spec.Selector"}
		}
		if err := hasImmutableLabelChange(existing.Spec.Selector.MatchLabels, desired.Spec.Template.Labels); err != nil {
			return err
		}
	}

	existing.Spec.MinReadySeconds = desired.Spec.MinReadySeconds
	existing.Spec.RevisionHistoryLimit = desired.Spec.RevisionHistoryLimit
	existing.Spec.UpdateStrategy = desired.Spec.UpdateStrategy

	if err := mutatePodTemplate(&existing.Spec.Template, &desired.Spec.Template); err != nil {
		return err
	}

	return nil
}

func mutateDeployment(existing, desired *appsv1.Deployment) error {
	if !existing.CreationTimestamp.IsZero() {
		if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
			return &ImmutableFieldChangeErr{Field: "Spec.Selector"}
		}
		if err := hasImmutableLabelChange(existing.Spec.Selector.MatchLabels, desired.Spec.Template.Labels); err != nil {
			return err
		}
	}

	existing.Spec.MinReadySeconds = desired.Spec.MinReadySeconds
	existing.Spec.Paused = desired.Spec.Paused
	existing.Spec.ProgressDeadlineSeconds = desired.Spec.ProgressDeadlineSeconds
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.RevisionHistoryLimit = desired.Spec.RevisionHistoryLimit
	existing.Spec.Strategy = desired.Spec.Strategy

	if err := mutatePodTemplate(&existing.Spec.Template, &desired.Spec.Template); err != nil {
		return err
	}

	return nil
}

func mutateStatefulSet(existing, desired *appsv1.StatefulSet) error {
	if !existing.CreationTimestamp.IsZero() {
		if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
			return &ImmutableFieldChangeErr{Field: "Spec.Selector"}
		}
		if err := hasImmutableLabelChange(existing.Spec.Selector.MatchLabels, desired.Spec.Template.Labels); err != nil {
			return err
		}
		if hasVolumeClaimsTemplatesChanged(existing, desired) {
			return &ImmutableFieldChangeErr{Field: "Spec.VolumeClaimTemplates"}
		}
	}

	existing.Spec.MinReadySeconds = desired.Spec.MinReadySeconds
	existing.Spec.Ordinals = desired.Spec.Ordinals
	existing.Spec.PersistentVolumeClaimRetentionPolicy = desired.Spec.PersistentVolumeClaimRetentionPolicy
	existing.Spec.PodManagementPolicy = desired.Spec.PodManagementPolicy
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.RevisionHistoryLimit = desired.Spec.RevisionHistoryLimit
	existing.Spec.ServiceName = desired.Spec.ServiceName
	existing.Spec.UpdateStrategy = desired.Spec.UpdateStrategy

	for i := range existing.Spec.VolumeClaimTemplates {
		existing.Spec.VolumeClaimTemplates[i].TypeMeta = desired.Spec.VolumeClaimTemplates[i].TypeMeta
		existing.Spec.VolumeClaimTemplates[i].ObjectMeta = desired.Spec.VolumeClaimTemplates[i].ObjectMeta
		existing.Spec.VolumeClaimTemplates[i].Spec = desired.Spec.VolumeClaimTemplates[i].Spec
	}

	if err := mutatePodTemplate(&existing.Spec.Template, &desired.Spec.Template); err != nil {
		return err
	}

	return nil
}

func mutateCertificate(existing, desired *cmv1.Certificate) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutateIssuer(existing, desired *cmv1.Issuer) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
}

func mutatePodTemplate(existing, desired *corev1.PodTemplateSpec) error {
	if err := mergeWithOverride(&existing.Labels, desired.Labels); err != nil {
		return err
	}

	if err := mergeWithOverride(&existing.Annotations, desired.Annotations); err != nil {
		return err
	}

	existing.Spec = desired.Spec

	return nil

}

func hasImmutableLabelChange(existingSelectorLabels, desiredLabels map[string]string) error {
	for k, v := range existingSelectorLabels {
		if vv, ok := desiredLabels[k]; !ok || vv != v {
			return &ImmutableFieldChangeErr{Field: "Spec.Template.Metadata.Labels"}
		}
	}
	return nil
}

// hasVolumeClaimsTemplatesChanged if volume claims template change has been detected.
// We need to do this manually due to some fields being automatically filled by the API server
// and these needs to be excluded from the comparison to prevent false positives.
func hasVolumeClaimsTemplatesChanged(existing, desired *appsv1.StatefulSet) bool {
	if len(desired.Spec.VolumeClaimTemplates) != len(existing.Spec.VolumeClaimTemplates) {
		return true
	}

	for i := range desired.Spec.VolumeClaimTemplates {
		// VolumeMode is automatically set by the API server, so if it is not set in the CR, assume it's the same as the existing one.
		if desired.Spec.VolumeClaimTemplates[i].Spec.VolumeMode == nil || *desired.Spec.VolumeClaimTemplates[i].Spec.VolumeMode == "" {
			desired.Spec.VolumeClaimTemplates[i].Spec.VolumeMode = existing.Spec.VolumeClaimTemplates[i].Spec.VolumeMode
		}

		if desired.Spec.VolumeClaimTemplates[i].Name != existing.Spec.VolumeClaimTemplates[i].Name {
			return true
		}
		if !apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Annotations, existing.Spec.VolumeClaimTemplates[i].Annotations) {
			return true
		}
		if !apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Labels, existing.Spec.VolumeClaimTemplates[i].Labels) {
			return true
		}
		if !apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Spec, existing.Spec.VolumeClaimTemplates[i].Spec) {
			return true
		}
	}

	return false
}
