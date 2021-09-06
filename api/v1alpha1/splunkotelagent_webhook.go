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

	ta "github.com/signalfx/splunk-otel-operator/pkg/targetallocator/adapters"
)

// log is for logging in this package.
var agentlog = logf.Log.WithName("splunkotelagent-resource")

func (r *SplunkOtelAgent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-splunk-com-v1alpha1-splunkotelagent,mutating=true,failurePolicy=fail,groups=splunk.com,resources=splunkotelagents,verbs=create;update,versions=v1alpha1,name=msplunkotelagent.kb.io,sideEffects=none,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &SplunkOtelAgent{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *SplunkOtelAgent) Default() {
	if len(r.Spec.Mode) == 0 {
		r.Spec.Mode = ModeDeployment
	}

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "splunk-otel-operator"
	}

	agentlog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-splunk-com-v1alpha1-splunkotelagent,mutating=false,failurePolicy=fail,groups=splunk.com,resources=splunkotelagents,versions=v1alpha1,name=vsplunkotelagentcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=delete,path=/validate-splunk-com-v1alpha1-splunkotelagent,mutating=false,failurePolicy=ignore,groups=splunk.com,resources=splunkotelagents,versions=v1alpha1,name=vsplunkotelagentdelete.kb.io,sideEffects=none,admissionReviewVersions=v1;v1beta1

var _ webhook.Validator = &SplunkOtelAgent{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *SplunkOtelAgent) ValidateCreate() error {
	agentlog.Info("validate create", "name", r.Name)
	return r.validateCRDSpec()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *SplunkOtelAgent) ValidateUpdate(old runtime.Object) error {
	agentlog.Info("validate update", "name", r.Name)
	return r.validateCRDSpec()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *SplunkOtelAgent) ValidateDelete() error {
	agentlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *SplunkOtelAgent) validateCRDSpec() error {
	// validate volumeClaimTemplates
	if r.Spec.Mode != ModeStatefulSet && len(r.Spec.VolumeClaimTemplates) > 0 {
		return fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'volumeClaimTemplates'", r.Spec.Mode)
	}

	// validate replicas
	if (r.Spec.Mode == ModeSidecar || r.Spec.Mode == ModeDaemonSet) && r.Spec.Replicas != nil {
		return fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'replicas'", r.Spec.Mode)
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

	return nil
}
