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

package collectorwebhook

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var _ admission.CustomValidator = &CollectorValidatingWebhook{}

type CollectorValidatingWebhook struct {
	logger logr.Logger
	c      client.Client
}

func (c CollectorValidatingWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	otelcol, ok := obj.(*v1alpha1.OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return otelcol.ValidateCRDSpec()
}

func (c CollectorValidatingWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	otelcol, ok := newObj.(*v1alpha1.OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", newObj)
	}
	return otelcol.ValidateCRDSpec()
}

func (c CollectorValidatingWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	otelcol, ok := obj.(*v1alpha1.OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return otelcol.ValidateCRDSpec()
}

func SetupCollectorValidatingWebhookWithManager(mgr controllerruntime.Manager) error {
	cvw := &CollectorValidatingWebhook{
		c:      mgr.GetClient(),
		logger: mgr.GetLogger().WithValues("handler", "CollectorValidatingWebhook"),
	}
	return controllerruntime.NewWebhookManagedBy(mgr).
		For(&v1alpha1.OpenTelemetryCollector{}).
		WithValidator(cvw).
		Complete()
}
