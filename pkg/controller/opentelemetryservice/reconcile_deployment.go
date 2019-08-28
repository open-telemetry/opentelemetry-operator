package opentelemetryservice

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileDeployment reconciles the deployment(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryService) reconcileDeployment(ctx context.Context) error {
	desired := deployment(ctx)
	r.setControllerReference(ctx, desired)

	expected := &appsv1.Deployment{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, expected)
	if err != nil && errors.IsNotFound(err) {
		if err := r.client.Create(ctx, desired); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// it exists already, merge the two if the end result isn't identical to the existing one
	// TODO(jpkroehling)

	return nil
}

func deployment(ctx context.Context) *appsv1.Deployment {
	instance := ctx.Value(opentelemetry.Instance).(*v1alpha1.OpenTelemetryService)
	name := fmt.Sprintf("%s-collector", instance.Name)

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	specAnnotations := instance.Annotations
	specAnnotations["prometheus.io/scrape"] = "true"
	specAnnotations["prometheus.io/port"] = "8888"
	specAnnotations["prometheus.io/path"] = "/metrics"

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
						Name:  "opentelemetry-service",
						Image: "quay.io/jpkroehling/opentelemetry-service:latest",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      name,
							MountPath: "/conf",
						}},
						Args: []string{
							fmt.Sprintf("--config=/conf/%s", opentelemetry.CollectorConfigMapEntry),
						},
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
