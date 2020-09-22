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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var opentelemetrycollectorlog = logf.Log.WithName("opentelemetrycollector-resource")

func (r *OpenTelemetryCollector) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=true,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=create;update,versions=v1alpha1,name=mopentelemetrycollector.kb.io

var _ webhook.Defaulter = &OpenTelemetryCollector{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenTelemetryCollector) Default() {
	if len(r.Spec.Mode) == 0 {
		r.Spec.Mode = ModeDeployment
	}

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	}

	opentelemetrycollectorlog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1alpha1,name=vopentelemetrycollector.kb.io

var _ webhook.Validator = &OpenTelemetryCollector{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenTelemetryCollector) ValidateCreate() error {
	opentelemetrycollectorlog.Info("validate create", "name", r.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenTelemetryCollector) ValidateUpdate(old runtime.Object) error {
	opentelemetrycollectorlog.Info("validate update", "name", r.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OpenTelemetryCollector) ValidateDelete() error {
	opentelemetrycollectorlog.Info("validate delete", "name", r.Name)
	return nil
}
