package manifests

import (
	"fmt"
	"reflect"

	apiequality "k8s.io/apimachinery/pkg/api/equality"

	"github.com/pkg/errors"

	"github.com/imdario/mergo"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MutateFuncFor returns a mutate function based on the
// existing resource's concrete type. It supports currently
// only the following types or else panics:
// - ConfigMap
// - Service
// - ServiceAccount
// - Deployment
// - DaemonSet
// - StatefulSet
// - Route
// - Secret.
func MutateFuncFor(existing, desired client.Object) controllerutil.MutateFn {
	return func() error {
		existingAnnotations := existing.GetAnnotations()
		if err := mergeWithOverride(&existingAnnotations, desired.GetAnnotations()); err != nil {
			return err
		}
		existing.SetAnnotations(existingAnnotations)

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
			return mutateService(svc, wantSvc)

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

		case *networkingv1.Ingress:
			ing := existing.(*networkingv1.Ingress)
			wantIng := desired.(*networkingv1.Ingress)
			mutateIngress(ing, wantIng)

		case *routev1.Route:
			rt := existing.(*routev1.Route)
			wantRt := desired.(*routev1.Route)
			mutateRoute(rt, wantRt)

		case *corev1.Secret:
			pr := existing.(*corev1.Secret)
			wantPr := desired.(*corev1.Secret)
			mutateSecret(pr, wantPr)

		default:
			t := reflect.TypeOf(existing).String()
			return errors.New(fmt.Sprintf("missing mutate implementation for resource type: %s", t))
		}
		return nil
	}
}

func mergeWithOverride(dst, src interface{}) error {
	err := mergo.Merge(dst, src, mergo.WithOverride)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to mergeWithOverride, dst: %v, src: %v", dst, src))
	}
	return nil
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
	existing.Annotations = desired.Annotations
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
}

func mutateRoleBinding(existing, desired *rbacv1.RoleBinding) {
	existing.Annotations = desired.Annotations
	existing.Labels = desired.Labels
	existing.Subjects = desired.Subjects
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

func mutateService(existing, desired *corev1.Service) error {
	existing.Spec.Ports = desired.Spec.Ports
	if err := mergeWithOverride(&existing.Spec.Selector, desired.Spec.Selector); err != nil {
		return err
	}
	return nil
}

func mutateDaemonset(existing, desired *appsv1.DaemonSet) error {
	// Daemonset selector is immutable so we set this value only if
	// a new object is going to be created
	if existing.CreationTimestamp.IsZero() {
		existing.Spec.Selector = desired.Spec.Selector
	}

	// TODO: is there anything wrong with doing this?
	if err := mergeWithOverride(&existing.Spec, desired.Spec); err != nil {
		return err
	}
	return nil
}

func mutateDeployment(existing, desired *appsv1.Deployment) error {
	// Deployment selector is immutable so we set this value only if
	// a new object is going to be created
	if existing.CreationTimestamp.IsZero() {
		existing.Spec.Selector = desired.Spec.Selector
	}
	existing.Spec.Replicas = desired.Spec.Replicas
	if err := mergeWithOverride(&existing.Spec.Template, desired.Spec.Template); err != nil {
		return err
	}
	if err := mergeWithOverride(&existing.Spec.Strategy, desired.Spec.Strategy); err != nil {
		return err
	}
	return nil
}

func mutateStatefulSet(existing, desired *appsv1.StatefulSet) error {
	if ok, field := hasImmutableFieldChange(existing, desired); !ok {
		return errors.New(fmt.Sprintf("attempting to mutate immutable field: %s", field))
	}
	// StatefulSet selector is immutable so we set this value only if
	// a new object is going to be created
	if existing.CreationTimestamp.IsZero() {
		existing.Spec.Selector = desired.Spec.Selector
	}
	existing.Spec.PodManagementPolicy = desired.Spec.PodManagementPolicy
	existing.Spec.Replicas = desired.Spec.Replicas
	// TODO: I don't think we can do this...
	//for i := range existing.Spec.VolumeClaimTemplates {
	//	existing.Spec.VolumeClaimTemplates[i].TypeMeta = desired.Spec.VolumeClaimTemplates[i].TypeMeta
	//	existing.Spec.VolumeClaimTemplates[i].ObjectMeta = desired.Spec.VolumeClaimTemplates[i].ObjectMeta
	//	existing.Spec.VolumeClaimTemplates[i].Spec = desired.Spec.VolumeClaimTemplates[i].Spec
	//}
	if err := mergeWithOverride(&existing.Spec.Template, desired.Spec.Template); err != nil {
		return err
	}
	return nil
}

func hasImmutableFieldChange(existing, desired *appsv1.StatefulSet) (bool, string) {
	if !apiequality.Semantic.DeepEqual(desired.Spec.Selector, existing.Spec.Selector) {
		return true, "Spec.Selector"
	}

	if hasVolumeClaimsTemplatesChanged(desired, existing) {
		return true, "Spec.VolumeClaimTemplates"
	}

	return false, ""
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
		if !apiequality.Semantic.DeepEqual(desired.Spec.VolumeClaimTemplates[i].Spec, existing.Spec.VolumeClaimTemplates[i].Spec) {
			return true
		}
	}

	return false
}
