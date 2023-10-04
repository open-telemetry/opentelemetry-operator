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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var opampbridgelog = logf.Log.WithName("opampbridge-resource")

// atleast these capabilities should be enabled.
var requiredCapabilities = []OpAMPBridgeCapability{
	OpAMPBridgeCapabilityAcceptsRemoteConfig,
	OpAMPBridgeCapabilityReportsEffectiveConfig,
	OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings,
	OpAMPBridgeCapabilityReportsHealth,
	OpAMPBridgeCapabilityReportsRemoteConfig,
}

func (r *OpAMPBridge) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-opampbridge,mutating=true,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=opampbridges,verbs=create;update,versions=v1alpha1,name=mopampbridge.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpAMPBridge{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *OpAMPBridge) Default() {
	opampbridgelog.Info("default", "name", r.Name)
	if len(r.Spec.UpgradeStrategy) == 0 {
		r.Spec.UpgradeStrategy = UpgradeStrategyAutomatic
	}

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	}

	one := int32(1)
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = &one
	}
}

//+kubebuilder:webhook:path=/validate-opentelemetry-io-v1alpha1-opampbridge,mutating=false,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=opampbridges,verbs=create;update;delete,versions=v1alpha1,name=vopampbridge.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OpAMPBridge{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpAMPBridge) ValidateCreate() (admission.Warnings, error) {
	opampbridgelog.Info("validate create", "name", r.Name)
	return nil, r.validateCRDSpec()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *OpAMPBridge) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	opampbridgelog.Info("validate update", "name", r.Name)
	return nil, r.validateCRDSpec()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *OpAMPBridge) ValidateDelete() (admission.Warnings, error) {
	opampbridgelog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *OpAMPBridge) validateCRDSpec() error {

	// check required fields

	if len(strings.TrimSpace(r.Spec.Endpoint)) == 0 {
		return fmt.Errorf("the OpAMP server endpoint is not specified")
	}

	if len(strings.TrimSpace(r.Spec.Protocol)) == 0 {
		return fmt.Errorf("the transport for OpAMP server protocol is not specified")
	}

	if len(r.Spec.Capabilities) == 0 {
		return fmt.Errorf("the capabilities supported by OpAMP Bridge are not specified")
	}

	// check if necessary capabilities are enabled
	for _, requiredCapability := range requiredCapabilities {
		enabled := false
		for _, enabledCapability := range r.Spec.Capabilities {
			if requiredCapability == enabledCapability {
				enabled = true
				break
			}
		}
		if !enabled {
			return fmt.Errorf("required capabilities must be enabled. Required capabilities: %s", requiredCapabilities)
		}
	}

	// validate port config
	for _, p := range r.Spec.Ports {
		nameErrs := validation.IsValidPortName(p.Name)
		numErrs := validation.IsValidPortNum(int(p.Port))
		if len(nameErrs) > 0 || len(numErrs) > 0 {
			return fmt.Errorf("the OpAMPBridge Spec Ports configuration is incorrect, port name '%s' errors: %s, num '%d' errors: %s",
				p.Name, nameErrs, p.Port, numErrs)
		}
	}

	// check for maximum replica count
	if r.Spec.Replicas != nil && *r.Spec.Replicas > 1 {
		return fmt.Errorf("replica count must not be greater than 1")
	}
	return nil
}
