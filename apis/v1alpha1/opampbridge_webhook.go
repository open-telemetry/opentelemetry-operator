// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var (
	_ admission.CustomValidator = &OpAMPBridgeWebhook{}
	_ admission.CustomDefaulter = &OpAMPBridgeWebhook{}
)

//+kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-opampbridge,mutating=true,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=opampbridges,verbs=create;update,versions=v1alpha1,name=mopampbridge.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-opentelemetry-io-v1alpha1-opampbridge,mutating=false,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=opampbridges,verbs=create;update,versions=v1alpha1,name=vopampbridgecreateupdate.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-opentelemetry-io-v1alpha1-opampbridge,mutating=false,failurePolicy=ignore,sideEffects=None,groups=opentelemetry.io,resources=opampbridges,verbs=delete,versions=v1alpha1,name=vopampbridgedelete.kb.io,admissionReviewVersions=v1
//+kubebuilder:object:generate=false

type OpAMPBridgeWebhook struct {
	logger logr.Logger
	cfg    config.Config
	scheme *runtime.Scheme
}

func (o *OpAMPBridgeWebhook) Default(ctx context.Context, obj runtime.Object) error {
	opampBridge, ok := obj.(*OpAMPBridge)
	if !ok {
		return fmt.Errorf("expected an OpAMPBridge, received %T", obj)
	}
	return o.defaulter(opampBridge)
}

func (o *OpAMPBridgeWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	opampBridge, ok := obj.(*OpAMPBridge)
	if !ok {
		return nil, fmt.Errorf("expected an OpAMPBridge, received %T", obj)
	}
	return o.validate(opampBridge)
}

func (o *OpAMPBridgeWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	opampBridge, ok := newObj.(*OpAMPBridge)
	if !ok {
		return nil, fmt.Errorf("expected an OpAMPBridge, received %T", newObj)
	}
	return o.validate(opampBridge)
}

func (o *OpAMPBridgeWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	opampBridge, ok := obj.(*OpAMPBridge)
	if !ok || opampBridge == nil {
		return nil, fmt.Errorf("expected an OpAMPBridge, received %T", obj)
	}
	return o.validate(opampBridge)
}

func (o *OpAMPBridgeWebhook) defaulter(r *OpAMPBridge) error {
	if len(r.Spec.UpgradeStrategy) == 0 {
		r.Spec.UpgradeStrategy = UpgradeStrategyAutomatic
	}

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	one := int32(1)
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = &one
	}

	// ReportsStatus Capability must be set
	if r.Spec.Capabilities == nil {
		r.Spec.Capabilities = make(map[OpAMPBridgeCapability]bool)
	}
	enabled, found := r.Spec.Capabilities[OpAMPBridgeCapabilityReportsStatus]
	if !enabled || !found {
		r.Spec.Capabilities[OpAMPBridgeCapabilityReportsStatus] = true
	}
	return nil
}

func (o *OpAMPBridgeWebhook) validate(r *OpAMPBridge) (admission.Warnings, error) {
	warnings := admission.Warnings{}

	// validate OpAMP server endpoint
	if len(strings.TrimSpace(r.Spec.Endpoint)) == 0 {
		return warnings, fmt.Errorf("the OpAMP server endpoint is not specified")
	}

	// validate OpAMPBridge capabilities
	if len(r.Spec.Capabilities) == 0 {
		return warnings, fmt.Errorf("the capabilities supported by OpAMP Bridge are not specified")
	}

	// validate port config
	for _, p := range r.Spec.Ports {
		nameErrs := validation.IsValidPortName(p.Name)
		numErrs := validation.IsValidPortNum(int(p.Port))
		if len(nameErrs) > 0 || len(numErrs) > 0 {
			return warnings, fmt.Errorf("the OpAMPBridge Spec Ports configuration is incorrect, port name '%s' errors: %s, num '%d' errors: %s",
				p.Name, nameErrs, p.Port, numErrs)
		}
	}

	// check for maximum replica count
	if r.Spec.Replicas != nil && *r.Spec.Replicas > 1 {
		return warnings, fmt.Errorf("replica count must not be greater than 1")
	}
	return warnings, nil
}

func SetupOpAMPBridgeWebhook(mgr ctrl.Manager, cfg config.Config) error {
	webhook := &OpAMPBridgeWebhook{
		logger: mgr.GetLogger().WithValues("handler", "OpAMPBridgeWebhook"),
		scheme: mgr.GetScheme(),
		cfg:    cfg,
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&OpAMPBridge{}).
		WithValidator(webhook).
		WithDefaulter(webhook).
		Complete()
}
