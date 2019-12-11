package opentelemetrycollector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileDeployment reconciles the deployment(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryCollector) reconcileDeployment(ctx context.Context) error {
	desired := deployments(ctx)

	// first, handle the create/update parts
	if err := r.reconcileExpectedDeployments(ctx, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected deployments: %v", err)
	}

	// then, delete the extra objects
	if err := r.deleteDeployments(ctx, desired); err != nil {
		return fmt.Errorf("failed to reconcile the deployments to be deleted: %v", err)
	}

	return nil
}

func deployments(ctx context.Context) []*appsv1.Deployment {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)

	var desired []*appsv1.Deployment
	if len(instance.Spec.Mode) == 0 || instance.Spec.Mode == opentelemetry.ModeDeployment {
		desired = append(desired, deployment(ctx))
	}

	return desired
}

func deployment(ctx context.Context) *appsv1.Deployment {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	name := resourceName(instance.Name)

	image := instance.Spec.Image
	if len(image) == 0 {
		image = viper.GetString(opentelemetry.OtelColImageConfigKey)
	}

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	specAnnotations := instance.Annotations
	if specAnnotations == nil {
		specAnnotations = map[string]string{}
	}

	specAnnotations["prometheus.io/scrape"] = "true"
	specAnnotations["prometheus.io/port"] = "8888"
	specAnnotations["prometheus.io/path"] = "/metrics"

	argsMap := instance.Spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}

	if _, exists := argsMap["config"]; exists {
		logger.Info("the 'config' flag isn't allowed and is being ignored")
	}

	// this effectively overrides any 'config' entry that might exist in the CR
	argsMap["config"] = fmt.Sprintf("/conf/%s", opentelemetry.CollectorConfigMapEntry)

	var args []string
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: specAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "opentelemetry-collector",
						Image: image,
						VolumeMounts: []corev1.VolumeMount{{
							Name:      name,
							MountPath: "/conf",
						}},
						Args: args,
					}},
					Volumes: []corev1.Volume{{
						Name: name,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: name},
								Items: []corev1.KeyToPath{{
									Key:  opentelemetry.CollectorConfigMapEntry,
									Path: opentelemetry.CollectorConfigMapEntry,
								}},
							},
						},
					}},
				},
			},
		},
	}
}

func (r *ReconcileOpenTelemetryCollector) reconcileExpectedDeployments(ctx context.Context, expected []*appsv1.Deployment) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	for _, obj := range expected {
		desired := obj
		r.setControllerReference(ctx, desired)

		existing := &appsv1.Deployment{}
		err := r.clients.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
		if err != nil && errors.IsNotFound(err) {
			if err := r.clients.client.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create: %v", err)
			}

			logger.WithValues("deployment.name", desired.Name, "deployment.namespace", desired.Namespace).V(2).Info("created")
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %v", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if err := r.clients.client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to apply changes: %v", err)
		}
		logger.V(2).Info("applied", "deployment.name", desired.Name, "deployment.namespace", desired.Namespace)
	}

	return nil
}

func (r *ReconcileOpenTelemetryCollector) deleteDeployments(ctx context.Context, expected []*appsv1.Deployment) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	opts := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", instance.Namespace, instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &appsv1.DeploymentList{}
	if err := r.clients.client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %v", err)
	}

	for _, existing := range list.Items {
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
			}
		}

		if del {
			if err := r.clients.client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %v", err)
			}
			logger.V(2).Info("deleted", "deployment.name", existing.Name, "deployment.namespace", existing.Namespace)
		}
	}

	return nil
}
