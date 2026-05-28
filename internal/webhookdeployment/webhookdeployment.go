// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package webhookdeployment manages the standalone auto-instrumentation webhook deployment.
// When EnableStandaloneWebhook is set, the operator creates a separate Deployment, Service, and
// MutatingWebhookConfiguration for pod mutation, allowing independent scaling of the webhook.
package webhookdeployment

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

const (
	// Resource names.
	webhookName = "opentelemetry-operator-auto-instrumentation-webhook"

	// Label values.
	componentWebhook = "auto-instrumentation-webhook"
	managedByValue   = "opentelemetry-operator"

	// Ports.
	webhookPort = 9443
	healthPort  = 8081

	// Paths.
	certMountPath = "/tmp/k8s-webhook-server/serving-certs"

	// Operator's MutatingWebhookConfiguration name (set by kustomize).
	operatorWebhookConfigName = "opentelemetry-operator-mutating-webhook-configuration"
)

// Params holds the parameters for creating webhook deployment resources.
type Params struct {
	Client    client.Client
	Config    config.Config
	Namespace string
}

// NewParams creates Params from the current environment and config.
func NewParams(c client.Client, cfg config.Config) (Params, error) {
	namespace, err := autodetectutils.GetOperatorNamespace()
	if err != nil {
		return Params{}, fmt.Errorf("failed to get operator namespace: %w", err)
	}

	if cfg.OperatorImage == "" {
		return Params{}, fmt.Errorf("operator image is not set (use --operator-image flag or RELATED_IMAGE_OPERATOR env var)")
	}

	return Params{
		Client:    c,
		Config:    cfg,
		Namespace: namespace,
	}, nil
}

// Reconcile creates or updates the webhook Deployment, Service, and MutatingWebhookConfiguration.
func Reconcile(ctx context.Context, params Params) error {
	logger := log.FromContext(ctx)

	if !params.Config.EnableStandaloneWebhook {
		return nil
	}

	logger.Info("Reconciling auto-instrumentation webhook deployment",
		"replicas", params.Config.StandaloneWebhookReplicas,
		"namespace", params.Namespace)

	if err := reconcileService(ctx, params); err != nil {
		return fmt.Errorf("failed to reconcile webhook service: %w", err)
	}

	if err := reconcileDeployment(ctx, params); err != nil {
		return fmt.Errorf("failed to reconcile webhook deployment: %w", err)
	}

	if err := reconcileMutatingWebhookConfiguration(ctx, params); err != nil {
		return fmt.Errorf("failed to reconcile mutating webhook configuration: %w", err)
	}

	// Remove mpod.kb.io from operator's webhook configuration since standalone handles it
	if err := removePodWebhookFromOperator(ctx, params); err != nil {
		return fmt.Errorf("failed to remove pod webhook from operator config: %w", err)
	}

	return nil
}

// labels returns the labels for webhook resources.
func labels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       webhookName,
		"app.kubernetes.io/instance":   componentWebhook,
		"app.kubernetes.io/component":  componentWebhook,
		"app.kubernetes.io/managed-by": managedByValue,
	}
}

// selectorLabels returns the selector labels for webhook pods.
func selectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      webhookName,
		"app.kubernetes.io/instance":  componentWebhook,
		"app.kubernetes.io/component": componentWebhook,
	}
}

func reconcileService(ctx context.Context, params Params) error {
	logger := log.FromContext(ctx)

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: params.Namespace,
			Labels:    labels(),
			Annotations: map[string]string{
				// OpenShift service serving certificates - auto-provisions TLS certs
				"service.beta.openshift.io/serving-cert-secret-name": webhookName + "-cert",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromInt32(webhookPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: selectorLabels(),
		},
	}

	existing := &corev1.Service{}
	err := params.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating webhook service", "name", desired.Name)
		return params.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update existing service
	existing.Labels = desired.Labels
	existing.Annotations = desired.Annotations
	existing.Spec.Ports = desired.Spec.Ports
	existing.Spec.Selector = desired.Spec.Selector
	logger.Info("Updating webhook service", "name", desired.Name)
	return params.Client.Update(ctx, existing)
}

func reconcileDeployment(ctx context.Context, params Params) error {
	logger := log.FromContext(ctx)

	replicas := params.Config.StandaloneWebhookReplicas
	if replicas == 0 {
		replicas = 1
	}
	certSecretName := webhookName + "-cert"

	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: params.Namespace,
			Labels:    labels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "opentelemetry-operator-controller-manager",
					Containers: []corev1.Container{
						{
							Name:    "webhook",
							Image:   params.Config.OperatorImage,
							Command: []string{"./manager"},
							Args:    buildArgs(params.Config),
							Ports: []corev1.ContainerPort{
								{
									Name:          "webhook-server",
									ContainerPort: webhookPort,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "health",
									ContainerPort: healthPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt32(healthPort),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.FromInt32(healthPort),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cert",
									MountPath: certMountPath,
									ReadOnly:  true,
								},
							},
							Env: buildEnvVars(params.Config),
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  certSecretName,
									DefaultMode: ptr.To[int32](420),
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			},
		},
	}

	existing := &appsv1.Deployment{}
	err := params.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating webhook deployment", "name", desired.Name, "replicas", replicas)
		return params.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update existing deployment
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
	logger.Info("Updating webhook deployment", "name", desired.Name, "replicas", replicas)
	return params.Client.Update(ctx, existing)
}

func reconcileMutatingWebhookConfiguration(ctx context.Context, params Params) error {
	logger := log.FromContext(ctx)

	scope := admissionregistrationv1.AllScopes

	desired := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   webhookName,
			Labels: labels(),
			Annotations: map[string]string{
				// OpenShift injects the CA bundle from the service serving cert
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "mpod.kb.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      webhookName,
						Namespace: params.Namespace,
						Path:      ptr.To("/mutate-v1-pod"),
						Port:      ptr.To[int32](443),
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &scope,
						},
					},
				},
				FailurePolicy:           ptr.To(admissionregistrationv1.Ignore),
				SideEffects:             ptr.To(admissionregistrationv1.SideEffectClassNone),
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}

	existing := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err := params.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating mutating webhook configuration", "name", desired.Name)
		return params.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update existing webhook configuration
	existing.Labels = desired.Labels
	existing.Annotations = desired.Annotations
	existing.Webhooks = desired.Webhooks
	logger.Info("Updating mutating webhook configuration", "name", desired.Name)
	return params.Client.Update(ctx, existing)
}

func buildArgs(cfg config.Config) []string {
	args := []string{
		"auto-instrumentation",
		fmt.Sprintf("--health-probe-addr=:%d", healthPort),
		fmt.Sprintf("--webhook-port=%d", webhookPort),
	}

	// Pass through instrumentation feature flags
	if cfg.EnableMultiInstrumentation {
		args = append(args, "--enable-multi-instrumentation=true")
	}
	if cfg.EnableApacheHttpdInstrumentation {
		args = append(args, "--enable-apache-httpd-instrumentation=true")
	}
	if cfg.EnableDotNetAutoInstrumentation {
		args = append(args, "--enable-dotnet-instrumentation=true")
	}
	if cfg.EnableGoAutoInstrumentation {
		args = append(args, "--enable-go-instrumentation=true")
	}
	if cfg.EnablePythonAutoInstrumentation {
		args = append(args, "--enable-python-instrumentation=true")
	}
	if cfg.EnableNginxAutoInstrumentation {
		args = append(args, "--enable-nginx-instrumentation=true")
	}
	if cfg.EnableNodeJSAutoInstrumentation {
		args = append(args, "--enable-nodejs-instrumentation=true")
	}
	if cfg.EnableJavaAutoInstrumentation {
		args = append(args, "--enable-java-instrumentation=true")
	}

	return args
}

func buildEnvVars(cfg config.Config) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	// Pass through auto-instrumentation images
	imageEnvVars := map[string]string{
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_JAVA":         cfg.AutoInstrumentationJavaImage,
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_NODEJS":       cfg.AutoInstrumentationNodeJSImage,
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_PYTHON":       cfg.AutoInstrumentationPythonImage,
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_DOTNET":       cfg.AutoInstrumentationDotNetImage,
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_GO":           cfg.AutoInstrumentationGoImage,
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_APACHE_HTTPD": cfg.AutoInstrumentationApacheHttpdImage,
		"RELATED_IMAGE_AUTO_INSTRUMENTATION_NGINX":        cfg.AutoInstrumentationNginxImage,
		"RELATED_IMAGE_COLLECTOR":                         cfg.CollectorImage,
	}

	for name, value := range imageEnvVars {
		if value != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  name,
				Value: value,
			})
		}
	}

	return envVars
}

// removePodWebhookFromOperator removes the mpod.kb.io webhook from the operator's
// MutatingWebhookConfiguration since the standalone webhook handles pod mutation.
func removePodWebhookFromOperator(ctx context.Context, params Params) error {
	logger := log.FromContext(ctx)

	existing := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err := params.Client.Get(ctx, client.ObjectKey{Name: operatorWebhookConfigName}, existing)
	if apierrors.IsNotFound(err) {
		// Operator webhook config doesn't exist yet, nothing to do
		return nil
	}
	if err != nil {
		return err
	}

	// Filter out the mpod.kb.io webhook
	var filteredWebhooks []admissionregistrationv1.MutatingWebhook
	removed := false
	for _, wh := range existing.Webhooks {
		if wh.Name == "mpod.kb.io" {
			removed = true
			continue
		}
		filteredWebhooks = append(filteredWebhooks, wh)
	}

	if !removed {
		// mpod.kb.io not found, nothing to do
		return nil
	}

	existing.Webhooks = filteredWebhooks
	logger.Info("Removing mpod.kb.io webhook from operator's webhook configuration", "name", operatorWebhookConfigName)
	return params.Client.Update(ctx, existing)
}
