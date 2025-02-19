// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

var (
	_                                  admission.CustomValidator = &InstrumentationWebhook{}
	_                                  admission.CustomDefaulter = &InstrumentationWebhook{}
	initContainerDefaultLimitResources                           = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	}
	initContainerDefaultRequestedResources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	}
)

// +kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-instrumentation,mutating=true,failurePolicy=fail,sideEffects=None,groups=opentelemetry.io,resources=instrumentations,verbs=create;update,versions=v1alpha1,name=minstrumentation.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1alpha1-instrumentation,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=instrumentations,versions=v1alpha1,name=vinstrumentationdelete.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:object:generate=false

type InstrumentationWebhook struct {
	logger logr.Logger
	cfg    config.Config
	scheme *runtime.Scheme
}

func (w InstrumentationWebhook) Default(ctx context.Context, obj runtime.Object) error {
	instrumentation, ok := obj.(*Instrumentation)
	if !ok {
		return fmt.Errorf("expected an Instrumentation, received %T", obj)
	}
	return w.defaulter(instrumentation)
}

func (w InstrumentationWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	inst, ok := obj.(*Instrumentation)
	if !ok {
		return nil, fmt.Errorf("expected an Instrumentation, received %T", obj)
	}
	return w.validate(inst)
}

func (w InstrumentationWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	inst, ok := newObj.(*Instrumentation)
	if !ok {
		return nil, fmt.Errorf("expected an Instrumentation, received %T", newObj)
	}
	return w.validate(inst)
}

func (w InstrumentationWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	inst, ok := obj.(*Instrumentation)
	if !ok || inst == nil {
		return nil, fmt.Errorf("expected an Instrumentation, received %T", obj)
	}
	return w.validate(inst)
}

func (w InstrumentationWebhook) defaulter(r *Instrumentation) error {
	if r.Labels == nil {
		r.Labels = map[string]string{}
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
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		}
	}
	if r.Spec.Python.Resources.Requests == nil {
		r.Spec.Python.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
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
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		}
	}
	if r.Spec.Go.Resources.Requests == nil {
		r.Spec.Go.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
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
	// Set the defaulting annotations
	if r.Annotations == nil {
		r.Annotations = map[string]string{}
	}
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationJava] = w.cfg.AutoInstrumentationJavaImage()
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationNodeJS] = w.cfg.AutoInstrumentationNodeJSImage()
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationPython] = w.cfg.AutoInstrumentationPythonImage()
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationDotNet] = w.cfg.AutoInstrumentationDotNetImage()
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationGo] = w.cfg.AutoInstrumentationGoImage()
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationApacheHttpd] = w.cfg.AutoInstrumentationApacheHttpdImage()
	r.Annotations[constants.AnnotationDefaultAutoInstrumentationNginx] = w.cfg.AutoInstrumentationNginxImage()
	return nil
}

func (w InstrumentationWebhook) validate(r *Instrumentation) (admission.Warnings, error) {
	var warnings []string
	switch r.Spec.Sampler.Type {
	case "":
		warnings = append(warnings, "sampler type not set")
	case TraceIDRatio, ParentBasedTraceIDRatio:
		if r.Spec.Sampler.Argument != "" {
			rate, err := strconv.ParseFloat(r.Spec.Sampler.Argument, 64)
			if err != nil {
				return warnings, fmt.Errorf("spec.sampler.argument is not a number: %s", r.Spec.Sampler.Argument)
			}
			if rate < 0 || rate > 1 {
				return warnings, fmt.Errorf("spec.sampler.argument should be in rage [0..1]: %s", r.Spec.Sampler.Argument)
			}
		}
	case JaegerRemote, ParentBasedJaegerRemote:
		// value is a comma separated list of endpoint, pollingIntervalMs, initialSamplingRate
		// Example: `endpoint=http://localhost:14250,pollingIntervalMs=5000,initialSamplingRate=0.25`
		if r.Spec.Sampler.Argument != "" {
			err := validateJaegerRemoteSamplerArgument(r.Spec.Sampler.Argument)

			if err != nil {
				return warnings, fmt.Errorf("spec.sampler.argument is not a valid argument for sampler %s: %w", r.Spec.Sampler.Type, err)
			}
		}
	case AlwaysOn, AlwaysOff, ParentBasedAlwaysOn, ParentBasedAlwaysOff, XRaySampler:
	default:
		return warnings, fmt.Errorf("spec.sampler.type is not valid: %s", r.Spec.Sampler.Type)
	}

	var err error
	err = validateInstrVolume(r.Spec.ApacheHttpd.VolumeClaimTemplate, r.Spec.ApacheHttpd.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.apachehttpd.volumeClaimTemplate and spec.apachehttpd.volumeSizeLimit cannot both be defined: %w", err)
	}
	err = validateInstrVolume(r.Spec.DotNet.VolumeClaimTemplate, r.Spec.DotNet.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.dotnet.volumeClaimTemplate and spec.dotnet.volumeSizeLimit cannot both be defined: %w", err)
	}
	err = validateInstrVolume(r.Spec.Go.VolumeClaimTemplate, r.Spec.Go.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.go.volumeClaimTemplate and spec.go.volumeSizeLimit cannot both be defined: %w", err)
	}
	err = validateInstrVolume(r.Spec.Java.VolumeClaimTemplate, r.Spec.Java.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.java.volumeClaimTemplate and spec.java.volumeSizeLimit cannot both be defined: %w", err)
	}
	err = validateInstrVolume(r.Spec.Nginx.VolumeClaimTemplate, r.Spec.Nginx.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.nginx.volumeClaimTemplate and spec.nginx.volumeSizeLimit cannot both be defined: %w", err)
	}
	err = validateInstrVolume(r.Spec.NodeJS.VolumeClaimTemplate, r.Spec.NodeJS.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.nodejs.volumeClaimTemplate and spec.nodejs.volumeSizeLimit cannot both be defined: %w", err)
	}
	err = validateInstrVolume(r.Spec.Python.VolumeClaimTemplate, r.Spec.Python.VolumeSizeLimit)
	if err != nil {
		return warnings, fmt.Errorf("spec.python.volumeClaimTemplate and spec.python.volumeSizeLimit cannot both be defined: %w", err)
	}

	warnings = append(warnings, validateExporter(r.Spec.Exporter)...)

	return warnings, nil
}

func validateExporter(exporter Exporter) []string {
	var warnings []string
	if exporter.TLS != nil {
		tls := exporter.TLS
		if tls.Key != "" && tls.Cert == "" || tls.Cert != "" && tls.Key == "" {
			warnings = append(warnings, "both exporter.tls.key and exporter.tls.cert mut be set")
		}

		if !strings.HasPrefix(exporter.Endpoint, "https://") {
			warnings = append(warnings, "exporter.tls is configured but exporter.endpoint is not enabling TLS with https://")
		}
	}
	if strings.HasPrefix(exporter.Endpoint, "https://") && exporter.TLS == nil {
		warnings = append(warnings, "exporter is using https:// but exporter.tls is unset")
	}

	return warnings
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

func validateInstrVolume(volumeClaimTemplate corev1.PersistentVolumeClaimTemplate, volumeSizeLimit *resource.Quantity) error {
	if !reflect.ValueOf(volumeClaimTemplate).IsZero() && volumeSizeLimit != nil {
		return fmt.Errorf("unable to resolve volume size")
	}
	return nil
}

func NewInstrumentationWebhook(logger logr.Logger, scheme *runtime.Scheme, cfg config.Config) *InstrumentationWebhook {
	return &InstrumentationWebhook{
		logger: logger,
		scheme: scheme,
		cfg:    cfg,
	}
}

func SetupInstrumentationWebhook(mgr ctrl.Manager, cfg config.Config) error {
	ivw := NewInstrumentationWebhook(
		mgr.GetLogger().WithValues("handler", "InstrumentationWebhook"),
		mgr.GetScheme(),
		cfg,
	)
	return ctrl.NewWebhookManagedBy(mgr).
		For(&Instrumentation{}).
		WithValidator(ivw).
		WithDefaulter(ivw).
		Complete()
}
