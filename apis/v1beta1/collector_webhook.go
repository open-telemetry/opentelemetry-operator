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

package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

var (
	_ admission.CustomValidator = &CollectorWebhook{}
	_ admission.CustomDefaulter = &CollectorWebhook{}
)

// +kubebuilder:webhook:path=/mutate-opentelemetry-io-v1beta1-opentelemetrycollector,mutating=true,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=create;update,versions=v1beta1,name=mopentelemetrycollectorbeta.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1beta1-opentelemetrycollector,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1beta1,name=vopentelemetrycollectorcreateupdatebeta.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1beta1-opentelemetrycollector,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1beta1,name=vopentelemetrycollectordeletebeta.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:object:generate=false

type CollectorWebhook struct {
	logger   logr.Logger
	cfg      config.Config
	scheme   *runtime.Scheme
	reviewer *rbac.Reviewer
}

func (c CollectorWebhook) Default(_ context.Context, obj runtime.Object) error {
	otelcol, ok := obj.(*OpenTelemetryCollector)
	if !ok {
		return fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return c.defaulter(otelcol)
}

func (c CollectorWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := obj.(*OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return c.validate(otelcol)
}

func (c CollectorWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := newObj.(*OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", newObj)
	}
	return c.validate(otelcol)
}

func (c CollectorWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := obj.(*OpenTelemetryCollector)
	if !ok || otelcol == nil {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return c.validate(otelcol)
}

func (c CollectorWebhook) defaulter(r *OpenTelemetryCollector) error {
	return nil
}

func (c CollectorWebhook) validate(r *OpenTelemetryCollector) (admission.Warnings, error) {
	warnings := admission.Warnings{}

	nullObjects := r.Spec.Config.nullObjects()
	if len(nullObjects) > 0 {
		warnings = append(warnings, fmt.Sprintf("Collector config spec.config has null objects: %s. For compatibility tooling (kustomize and kubectl edit) it is recommended to use empty obejects e.g. batch: {}.", strings.Join(nullObjects, ", ")))
	}
	return warnings, nil
}

func SetupCollectorWebhook(mgr ctrl.Manager, cfg config.Config, reviewer *rbac.Reviewer) error {
	cvw := &CollectorWebhook{
		reviewer: reviewer,
		logger:   mgr.GetLogger().WithValues("handler", "CollectorWebhook", "version", "v1beta1"),
		scheme:   mgr.GetScheme(),
		cfg:      cfg,
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&OpenTelemetryCollector{}).
		WithValidator(cvw).
		WithDefaulter(cvw).
		Complete()
}
