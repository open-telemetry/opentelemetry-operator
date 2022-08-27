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

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

// log is for logging in this package.
var opentelemetrycollectorlog = logf.Log.WithName("opentelemetrycollector-resource")

func (r *OpenTelemetryCollector) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=true,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=create;update,versions=v1alpha1,name=mopentelemetrycollector.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenTelemetryCollector{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *OpenTelemetryCollector) Default() {
	opentelemetrycollectorlog.Info("default", "name", r.Name)

	if len(r.Spec.Mode) == 0 {
		r.Spec.Mode = ModeDeployment
	}
	if len(r.Spec.UpgradeStrategy) == 0 {
		r.Spec.UpgradeStrategy = UpgradeStrategyAutomatic
	}

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	}

	// We can default to one because dependent objects Deployment and HorizontalPodAutoScaler
	// default to 1 as well.
	one := int32(1)
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = &one
	}
	if r.Spec.TargetAllocator.Enabled && r.Spec.TargetAllocator.Replicas == nil {
		r.Spec.TargetAllocator.Replicas = &one
	}

	// Set default targetCPUUtilization for autoscaler
	if r.Spec.MaxReplicas != nil && r.Spec.Autoscaler.TargetCPUUtilization == nil {
		defaultCPUTarget := int32(90)
		r.Spec.Autoscaler.TargetCPUUtilization = &defaultCPUTarget
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1alpha1,name=vopentelemetrycollectorcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1alpha1,name=vopentelemetrycollectordelete.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &OpenTelemetryCollector{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenTelemetryCollector) ValidateCreate() error {
	opentelemetrycollectorlog.Info("validate create", "name", r.Name)
	return r.validateCRDSpec()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenTelemetryCollector) ValidateUpdate(old runtime.Object) error {
	opentelemetrycollectorlog.Info("validate update", "name", r.Name)
	return r.validateCRDSpec()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *OpenTelemetryCollector) ValidateDelete() error {
	opentelemetrycollectorlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *OpenTelemetryCollector) validateCRDSpec() error {
	// validate volumeClaimTemplates
	if r.Spec.Mode != ModeStatefulSet && len(r.Spec.VolumeClaimTemplates) > 0 {
		return fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'volumeClaimTemplates'", r.Spec.Mode)
	}

	// validate tolerations
	if r.Spec.Mode == ModeSidecar && len(r.Spec.Tolerations) > 0 {
		return fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'tolerations'", r.Spec.Mode)
	}

	// validate target allocation
	if r.Spec.TargetAllocator.Enabled && r.Spec.Mode != ModeStatefulSet {
		return fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the target allocation deployment", r.Spec.Mode)
	}

	// validate Prometheus config for target allocation
	if r.Spec.TargetAllocator.Enabled {
		_, err := ta.ConfigToPromConfig(r.Spec.Config)
		if err != nil {
			return fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %s", err)
		}
	}

	// validate autoscale with horizontal pod autoscaler
	if r.Spec.MaxReplicas != nil {
		if *r.Spec.MaxReplicas < int32(1) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, maxReplicas should be defined and more than one")
		}

		if r.Spec.Replicas != nil && *r.Spec.Replicas > *r.Spec.MaxReplicas {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, replicas must not be greater than maxReplicas")
		}

		if r.Spec.MinReplicas != nil && *r.Spec.MinReplicas > *r.Spec.MaxReplicas {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas must not be greater than maxReplicas")
		}

		if r.Spec.MinReplicas != nil && *r.Spec.MinReplicas < int32(1) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas should be one or more")
		}

		if r.Spec.Autoscaler != nil && r.Spec.Autoscaler.Behavior != nil {
			if r.Spec.Autoscaler.Behavior.ScaleDown != nil && *r.Spec.Autoscaler.Behavior.ScaleDown.StabilizationWindowSeconds < int32(1) {
				return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, scaleDown should be one or more")
			}

			if r.Spec.Autoscaler.Behavior.ScaleUp != nil && *r.Spec.Autoscaler.Behavior.ScaleUp.StabilizationWindowSeconds < int32(1) {
				return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, scaleUp should be one or more")
			}
		}
		if r.Spec.Autoscaler.TargetCPUUtilization != nil && (*r.Spec.Autoscaler.TargetCPUUtilization < int32(1) || *r.Spec.Autoscaler.TargetCPUUtilization > int32(99)) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, targetCPUUtilization should be greater than 0 and less than 100")
		}

	}

	return nil
}
