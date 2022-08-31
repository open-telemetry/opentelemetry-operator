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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	AnnotationDefaultAutoInstrumentationJava   = "instrumentation.opentelemetry.io/default-auto-instrumentation-java-image"
	AnnotationDefaultAutoInstrumentationNodeJS = "instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image"
	AnnotationDefaultAutoInstrumentationPython = "instrumentation.opentelemetry.io/default-auto-instrumentation-python-image"
	AnnotationDefaultAutoInstrumentationDotNet = "instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image"
	envPrefix                                  = "OTEL_"
	envSplunkPrefix                            = "SPLUNK_"
)

// log is for logging in this package.
var instrumentationlog = logf.Log.WithName("instrumentation-resource")

func (r *Instrumentation) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-instrumentation,mutating=true,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=instrumentations,verbs=create;update,versions=v1alpha1,name=minstrumentation.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Instrumentation{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *Instrumentation) Default() {
	instrumentationlog.Info("default", "name", r.Name)
	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	}

	if r.Spec.Java.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationJava]; ok {
			r.Spec.Java.Image = val
		}
	}
	if r.Spec.NodeJS.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationNodeJS]; ok {
			r.Spec.NodeJS.Image = val
		}
	}
	if r.Spec.Python.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationPython]; ok {
			r.Spec.Python.Image = val
		}
	}
	if r.Spec.DotNet.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationDotNet]; ok {
			r.Spec.DotNet.Image = val
		}
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationdelete.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Instrumentation{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (in *Instrumentation) ValidateCreate() error {
	instrumentationlog.Info("validate create", "name", in.Name)
	return in.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (in *Instrumentation) ValidateUpdate(old runtime.Object) error {
	instrumentationlog.Info("validate update", "name", in.Name)
	return in.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (in *Instrumentation) ValidateDelete() error {
	instrumentationlog.Info("validate delete", "name", in.Name)
	return nil
}

func (in *Instrumentation) validate() error {
	switch in.Spec.Sampler.Type {
	case TraceIDRatio, ParentBasedTraceIDRatio:
		if in.Spec.Sampler.Argument != "" {
			rate, err := strconv.ParseFloat(in.Spec.Sampler.Argument, 64)
			if err != nil {
				return fmt.Errorf("spec.sampler.argument is not a number: %s", in.Spec.Sampler.Argument)
			}
			if rate < 0 || rate > 1 {
				return fmt.Errorf("spec.sampler.argument should be in rage [0..1]: %s", in.Spec.Sampler.Argument)
			}
		}
	case AlwaysOn, AlwaysOff, JaegerRemote, ParentBasedAlwaysOn, ParentBasedAlwaysOff, XRaySampler:
	}

	// validate env vars
	if err := in.validateEnv(in.Spec.Env); err != nil {
		return err
	}
	if err := in.validateEnv(in.Spec.Java.Env); err != nil {
		return err
	}
	if err := in.validateEnv(in.Spec.NodeJS.Env); err != nil {
		return err
	}
	if err := in.validateEnv(in.Spec.Python.Env); err != nil {
		return err
	}
	if err := in.validateEnv(in.Spec.DotNet.Env); err != nil {
		return err
	}

	return nil
}

func (in *Instrumentation) validateEnv(envs []corev1.EnvVar) error {
	for _, env := range envs {
		if !strings.HasPrefix(env.Name, envPrefix) && !strings.HasPrefix(env.Name, envSplunkPrefix) {
			return fmt.Errorf("env name should start with \"OTEL_\" or \"SPLUNK_\": %s", env.Name)
		}
	}
	return nil
}
