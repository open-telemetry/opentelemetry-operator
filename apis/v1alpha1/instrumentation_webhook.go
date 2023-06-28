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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	AnnotationDefaultAutoInstrumentationJava        = "instrumentation.opentelemetry.io/default-auto-instrumentation-java-image"
	AnnotationDefaultAutoInstrumentationNodeJS      = "instrumentation.opentelemetry.io/default-auto-instrumentation-nodejs-image"
	AnnotationDefaultAutoInstrumentationPython      = "instrumentation.opentelemetry.io/default-auto-instrumentation-python-image"
	AnnotationDefaultAutoInstrumentationDotNet      = "instrumentation.opentelemetry.io/default-auto-instrumentation-dotnet-image"
	AnnotationDefaultAutoInstrumentationGo          = "instrumentation.opentelemetry.io/default-auto-instrumentation-go-image"
	AnnotationDefaultAutoInstrumentationApacheHttpd = "instrumentation.opentelemetry.io/default-auto-instrumentation-apache-httpd-image"
	AnnotationDefaultAutoInstrumentationNginx       = "instrumentation.opentelemetry.io/default-auto-instrumentation-nginx-image"
	envPrefix                                       = "OTEL_"
	envSplunkPrefix                                 = "SPLUNK_"
)

// log is for logging in this package.
var instrumentationlog = logf.Log.WithName("instrumentation-resource")

var initContainerDefaultLimitResources = corev1.ResourceList{
	corev1.ResourceCPU:    resource.MustParse("500m"),
	corev1.ResourceMemory: resource.MustParse("128Mi"),
}
var initContainerDefaultRequestedResources = corev1.ResourceList{
	corev1.ResourceCPU:    resource.MustParse("1m"),
	corev1.ResourceMemory: resource.MustParse("128Mi"),
}

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
	if r.Spec.Java.Resources.Limits == nil {
		r.Spec.Java.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		}
	}
	if r.Spec.Java.Resources.Requests == nil {
		r.Spec.Java.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		}
	}
	if r.Spec.NodeJS.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationNodeJS]; ok {
			r.Spec.NodeJS.Image = val
		}
	}
	if r.Spec.NodeJS.Resources.Limits == nil {
		r.Spec.NodeJS.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
	}
	if r.Spec.NodeJS.Resources.Requests == nil {
		r.Spec.NodeJS.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
	}
	if r.Spec.Python.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationPython]; ok {
			r.Spec.Python.Image = val
		}
	}
	if r.Spec.Python.Resources.Limits == nil {
		r.Spec.Python.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		}
	}
	if r.Spec.Python.Resources.Requests == nil {
		r.Spec.Python.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		}
	}
	if r.Spec.DotNet.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationDotNet]; ok {
			r.Spec.DotNet.Image = val
		}
	}
	if r.Spec.DotNet.Resources.Limits == nil {
		r.Spec.DotNet.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
	}
	if r.Spec.DotNet.Resources.Requests == nil {
		r.Spec.DotNet.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
	}
	if r.Spec.Go.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationGo]; ok {
			r.Spec.Go.Image = val
		}
	}
	if r.Spec.Go.Resources.Limits == nil {
		r.Spec.Go.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		}
	}
	if r.Spec.Go.Resources.Requests == nil {
		r.Spec.Go.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		}
	}
	if r.Spec.ApacheHttpd.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationApacheHttpd]; ok {
			r.Spec.ApacheHttpd.Image = val
		}
	}
	if r.Spec.ApacheHttpd.Resources.Limits == nil {
		r.Spec.ApacheHttpd.Resources.Limits = initContainerDefaultLimitResources
	}
	if r.Spec.ApacheHttpd.Resources.Requests == nil {
		r.Spec.ApacheHttpd.Resources.Requests = initContainerDefaultRequestedResources
	}
	if r.Spec.ApacheHttpd.Version == "" {
		r.Spec.ApacheHttpd.Version = "2.4"
	}
	if r.Spec.ApacheHttpd.ConfigPath == "" {
		r.Spec.ApacheHttpd.ConfigPath = "/usr/local/apache2/conf"
	}
	if r.Spec.Nginx.Image == "" {
		if val, ok := r.Annotations[AnnotationDefaultAutoInstrumentationNginx]; ok {
			r.Spec.Nginx.Image = val
		}
	}
	if r.Spec.Nginx.Resources.Limits == nil {
		r.Spec.Nginx.Resources.Limits = initContainerDefaultLimitResources
	}
	if r.Spec.Nginx.Resources.Requests == nil {
		r.Spec.Nginx.Resources.Requests = initContainerDefaultRequestedResources
	}
	if r.Spec.Nginx.ConfigFile == "" {
		r.Spec.Nginx.ConfigFile = "/etc/nginx/nginx.conf"
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationdelete.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Instrumentation{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *Instrumentation) ValidateCreate() (admission.Warnings, error) {
	instrumentationlog.Info("validate create", "name", r.Name)
	return nil, r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *Instrumentation) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	instrumentationlog.Info("validate update", "name", r.Name)
	return nil, r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *Instrumentation) ValidateDelete() (admission.Warnings, error) {
	instrumentationlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func validateJaegerRemoteSamplerArgument(argument string) error {
	parts := strings.Split(argument, ",")

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid argument: %s, the argument should be in the form of key=value", part)
		}

		switch kv[0] {
		case "endpoint":
			if kv[1] == "" {
				return fmt.Errorf("endpoint cannot be empty")
			}
		case "pollingIntervalMs":
			if _, err := strconv.Atoi(kv[1]); err != nil {
				return fmt.Errorf("invalid pollingIntervalMs: %s", kv[1])
			}
		case "initialSamplingRate":
			rate, err := strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return fmt.Errorf("invalid initialSamplingRate: %s", kv[1])
			}
			if rate < 0 || rate > 1 {
				return fmt.Errorf("initialSamplingRate should be in rage [0..1]: %s", kv[1])
			}
		}
	}
	return nil
}

func (r *Instrumentation) validate() error {
	switch r.Spec.Sampler.Type {
	case TraceIDRatio, ParentBasedTraceIDRatio:
		if r.Spec.Sampler.Argument != "" {
			rate, err := strconv.ParseFloat(r.Spec.Sampler.Argument, 64)
			if err != nil {
				return fmt.Errorf("spec.sampler.argument is not a number: %s", r.Spec.Sampler.Argument)
			}
			if rate < 0 || rate > 1 {
				return fmt.Errorf("spec.sampler.argument should be in rage [0..1]: %s", r.Spec.Sampler.Argument)
			}
		}
	case JaegerRemote, ParentBasedJaegerRemote:
		// value is a comma separated list of endpoint, pollingIntervalMs, initialSamplingRate
		// Example: `endpoint=http://localhost:14250,pollingIntervalMs=5000,initialSamplingRate=0.25`
		if r.Spec.Sampler.Argument != "" {
			err := validateJaegerRemoteSamplerArgument(r.Spec.Sampler.Argument)

			if err != nil {
				return fmt.Errorf("spec.sampler.argument is not a valid argument for sampler %s: %w", r.Spec.Sampler.Type, err)
			}
		}
	case AlwaysOn, AlwaysOff, ParentBasedAlwaysOn, ParentBasedAlwaysOff, XRaySampler:
	default:
		return fmt.Errorf("spec.sampler.type is not valid: %s", r.Spec.Sampler.Type)
	}

	// validate env vars
	if err := r.validateEnv(r.Spec.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.Java.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.NodeJS.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.Python.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.DotNet.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.Go.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.ApacheHttpd.Env); err != nil {
		return err
	}
	if err := r.validateEnv(r.Spec.Nginx.Env); err != nil {
		return err
	}

	return nil
}

func (r *Instrumentation) validateEnv(envs []corev1.EnvVar) error {
	for _, env := range envs {
		if !strings.HasPrefix(env.Name, envPrefix) && !strings.HasPrefix(env.Name, envSplunkPrefix) {
			return fmt.Errorf("env name should start with \"OTEL_\" or \"SPLUNK_\": %s", env.Name)
		}
	}
	return nil
}
