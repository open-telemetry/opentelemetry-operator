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

package instrumentation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
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

var (
	_                                  admission.CustomValidator = &Webhook{}
	_                                  admission.CustomDefaulter = &Webhook{}
	initContainerDefaultLimitResources                           = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	}
	initContainerDefaultRequestedResources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	}
)

//+kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-instrumentation,mutating=true,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=instrumentations,verbs=create;update,versions=v1alpha1,name=minstrumentation.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationdelete.kb.io,sideEffects=none,admissionReviewVersions=v1

// Webhook is isolated because there are known registration issues when a custom webhook is in the same package
// as the types.
// See here: https://github.com/kubernetes-sigs/controller-runtime/issues/780#issuecomment-713408479
type Webhook struct {
	logger logr.Logger
	cfg    config.Config
	scheme *runtime.Scheme
}

func (w Webhook) Default(ctx context.Context, obj runtime.Object) error {
	instrumentation, ok := obj.(*v1alpha1.Instrumentation)
	if !ok {
		return fmt.Errorf("expected an Instrumentation, received %T", obj)
	}
	return w.defaulter(instrumentation)
}

func (w Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	inst, ok := obj.(*v1alpha1.Instrumentation)
	if !ok {
		return nil, fmt.Errorf("expected an Instrumentation, received %T", obj)
	}
	return w.validate(inst)
}

func (w Webhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	inst, ok := newObj.(*v1alpha1.Instrumentation)
	if !ok {
		return nil, fmt.Errorf("expected an Instrumentation, received %T", newObj)
	}
	return w.validate(inst)
}

func (w Webhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	inst, ok := obj.(*v1alpha1.Instrumentation)
	if !ok || inst == nil {
		return nil, fmt.Errorf("expected an Instrumentation, received %T", obj)
	}
	return w.validate(inst)
}

func (w Webhook) defaulter(r *v1alpha1.Instrumentation) error {
	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	}

	if r.Spec.Java.Image == "" {
		r.Spec.Java.Image = w.cfg.AutoInstrumentationJavaImage()
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
		r.Spec.NodeJS.Image = w.cfg.AutoInstrumentationNodeJSImage()
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
		r.Spec.Python.Image = w.cfg.AutoInstrumentationPythonImage()
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
		r.Spec.DotNet.Image = w.cfg.AutoInstrumentationDotNetImage()
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
		r.Spec.Go.Image = w.cfg.AutoInstrumentationGoImage()
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
		r.Spec.ApacheHttpd.Image = w.cfg.AutoInstrumentationApacheHttpdImage()
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
		r.Spec.Nginx.Image = w.cfg.AutoInstrumentationNginxImage()
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
	return nil
}

func (w Webhook) validate(r *v1alpha1.Instrumentation) (admission.Warnings, error) {
	var warnings []string
	switch r.Spec.Sampler.Type {
	case "":
		warnings = append(warnings, "sampler type not set")
	case v1alpha1.TraceIDRatio, v1alpha1.ParentBasedTraceIDRatio:
		if r.Spec.Sampler.Argument != "" {
			rate, err := strconv.ParseFloat(r.Spec.Sampler.Argument, 64)
			if err != nil {
				return warnings, fmt.Errorf("spec.sampler.argument is not a number: %s", r.Spec.Sampler.Argument)
			}
			if rate < 0 || rate > 1 {
				return warnings, fmt.Errorf("spec.sampler.argument should be in rage [0..1]: %s", r.Spec.Sampler.Argument)
			}
		}
	case v1alpha1.JaegerRemote, v1alpha1.ParentBasedJaegerRemote:
		// value is a comma separated list of endpoint, pollingIntervalMs, initialSamplingRate
		// Example: `endpoint=http://localhost:14250,pollingIntervalMs=5000,initialSamplingRate=0.25`
		if r.Spec.Sampler.Argument != "" {
			err := validateJaegerRemoteSamplerArgument(r.Spec.Sampler.Argument)

			if err != nil {
				return warnings, fmt.Errorf("spec.sampler.argument is not a valid argument for sampler %s: %w", r.Spec.Sampler.Type, err)
			}
		}
	case v1alpha1.AlwaysOn, v1alpha1.AlwaysOff, v1alpha1.ParentBasedAlwaysOn, v1alpha1.ParentBasedAlwaysOff, v1alpha1.XRaySampler:
	default:
		return warnings, fmt.Errorf("spec.sampler.type is not valid: %s", r.Spec.Sampler.Type)
	}

	// validate env vars
	if err := w.validateEnv(r.Spec.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.Java.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.NodeJS.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.Python.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.DotNet.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.Go.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.ApacheHttpd.Env); err != nil {
		return warnings, err
	}
	if err := w.validateEnv(r.Spec.Nginx.Env); err != nil {
		return warnings, err
	}
	return warnings, nil
}

func (w Webhook) validateEnv(envs []corev1.EnvVar) error {
	for _, env := range envs {
		if !strings.HasPrefix(env.Name, envPrefix) && !strings.HasPrefix(env.Name, envSplunkPrefix) {
			return fmt.Errorf("env name should start with \"OTEL_\" or \"SPLUNK_\": %s", env.Name)
		}
	}
	return nil
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

func NewInstrumentationWebhook(mgr ctrl.Manager, cfg config.Config) *Webhook {
	return &Webhook{
		logger: mgr.GetLogger().WithValues("handler", "InstrumentationWebhook"),
		scheme: mgr.GetScheme(),
		cfg:    cfg,
	}
}

func SetupInstrumentationValidatingWebhookWithManager(mgr ctrl.Manager, cfg config.Config) error {
	ivw := NewInstrumentationWebhook(mgr, cfg)
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Instrumentation{}).
		WithValidator(ivw).
		WithDefaulter(ivw).
		Complete()
}
