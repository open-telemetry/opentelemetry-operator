// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/instrumentation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

type autoInstConfig struct {
	id      string
	enabled bool
}

type InstrumentationUpgrade struct {
	Client                     client.Client
	Logger                     logr.Logger
	Recorder                   record.EventRecorder
	DefaultAutoInstJava        string
	DefaultAutoInstNodeJS      string
	DefaultAutoInstPython      string
	DefaultAutoInstDotNet      string
	DefaultAutoInstApacheHttpd string
	DefaultAutoInstNginx       string
	DefaultAutoInstGo          string
	defaultAnnotationToConfig  map[string]autoInstConfig
}

func NewInstrumentationUpgrade(client client.Client, logger logr.Logger, recorder record.EventRecorder, cfg config.Config) *InstrumentationUpgrade {
	defaultAnnotationToConfig := map[string]autoInstConfig{
		constants.AnnotationDefaultAutoInstrumentationApacheHttpd: {"enable-apache-httpd-instrumentation", cfg.EnableApacheHttpdInstrumentation},
		constants.AnnotationDefaultAutoInstrumentationDotNet:      {"enable-dotnet-instrumentation", cfg.EnableDotNetAutoInstrumentation},
		constants.AnnotationDefaultAutoInstrumentationGo:          {"enable-go-instrumentation", cfg.EnableGoAutoInstrumentation},
		constants.AnnotationDefaultAutoInstrumentationNginx:       {"enable-nginx-instrumentation", cfg.EnableNginxAutoInstrumentation},
		constants.AnnotationDefaultAutoInstrumentationPython:      {"enable-python-instrumentation", cfg.EnablePythonAutoInstrumentation},
		constants.AnnotationDefaultAutoInstrumentationNodeJS:      {"enable-nodejs-instrumentation", cfg.EnableNodeJSAutoInstrumentation},
		constants.AnnotationDefaultAutoInstrumentationJava:        {"enable-java-instrumentation", cfg.EnableJavaAutoInstrumentation},
	}

	return &InstrumentationUpgrade{
		Client:                     client,
		Logger:                     logger,
		DefaultAutoInstJava:        cfg.AutoInstrumentationJavaImage,
		DefaultAutoInstNodeJS:      cfg.AutoInstrumentationNodeJSImage,
		DefaultAutoInstPython:      cfg.AutoInstrumentationPythonImage,
		DefaultAutoInstDotNet:      cfg.AutoInstrumentationDotNetImage,
		DefaultAutoInstGo:          cfg.AutoInstrumentationGoImage,
		DefaultAutoInstApacheHttpd: cfg.AutoInstrumentationApacheHttpdImage,
		DefaultAutoInstNginx:       cfg.AutoInstrumentationNginxImage,
		Recorder:                   recorder,
		defaultAnnotationToConfig:  defaultAnnotationToConfig,
	}

}

// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch;update;patch

// ManagedInstances upgrades managed instances by the opentelemetry-operator.
func (u *InstrumentationUpgrade) ManagedInstances(ctx context.Context, cfg config.Config) error {
	list := &v1alpha1.InstrumentationList{}
	if cfg.EnableInstrumentationCRDs {
		u.Logger.Info("looking for managed Instrumentation instances to upgrade")
		if err := u.Client.List(ctx, list); err != nil {
			return fmt.Errorf("failed to list: %w", err)
		}
	} else {
		list.Items = []v1alpha1.Instrumentation{}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.Java); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-java-instrumentation": strconv.FormatBool(cfg.EnableJavaAutoInstrumentation),
					},
				},
			})
		}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.NodeJS); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-nodejs-instrumentation": strconv.FormatBool(cfg.EnableNodeJSAutoInstrumentation),
					},
				},
			})
		}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.DotNet); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-dotnet-instrumentation": strconv.FormatBool(cfg.EnableDotNetAutoInstrumentation),
					},
				},
			})
		}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.Go); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-go-instrumentation": strconv.FormatBool(cfg.EnableGoAutoInstrumentation),
					},
				},
			})
		}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.Python); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-python-instrumentation": strconv.FormatBool(cfg.EnablePythonAutoInstrumentation),
					},
				},
			})
		}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.ApacheHttpd); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-apache-httpd-instrumentation": strconv.FormatBool(cfg.EnableApacheHttpdInstrumentation),
					},
				},
			})
		}
		if instr := instrumentation.ConvertConfig(cfg.Instrumentation.Nginx); instr != nil {
			list.Items = append(list.Items, v1alpha1.Instrumentation{
				Status:   v1alpha1.InstrumentationStatus{},
				TypeMeta: metav1.TypeMeta{},
				Spec:     *instr,
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"enable-nginx-instrumentation": strconv.FormatBool(cfg.EnableNginxAutoInstrumentation),
					},
				},
			})
		}
	}
	for i := range list.Items {
		toUpgrade := list.Items[i]
		upgraded := u.upgrade(ctx, toUpgrade)
		if !reflect.DeepEqual(upgraded, toUpgrade) {
			// use update instead of patch because the patch does not upgrade annotations
			if err := u.Client.Update(ctx, upgraded); err != nil {
				u.Logger.Error(err, "failed to apply changes to instance", "name", upgraded.Name, "namespace", upgraded.Namespace)
				continue
			}
		}
	}

	if len(list.Items) == 0 {
		u.Logger.Info("no instances to upgrade")
	}
	return nil
}

func (u *InstrumentationUpgrade) upgrade(_ context.Context, inst v1alpha1.Instrumentation) *v1alpha1.Instrumentation {
	upgraded := inst.DeepCopy()
	for annotation, instCfg := range u.defaultAnnotationToConfig {
		autoInst := upgraded.Annotations[annotation]
		if autoInst != "" {
			if instCfg.enabled {
				switch annotation {
				case constants.AnnotationDefaultAutoInstrumentationApacheHttpd:
					if inst.Spec.ApacheHttpd.Image == autoInst {
						upgraded.Spec.ApacheHttpd.Image = u.DefaultAutoInstApacheHttpd
						upgraded.Annotations[annotation] = u.DefaultAutoInstApacheHttpd
					}
				case constants.AnnotationDefaultAutoInstrumentationDotNet:
					if inst.Spec.DotNet.Image == autoInst {
						upgraded.Spec.DotNet.Image = u.DefaultAutoInstDotNet
						upgraded.Annotations[annotation] = u.DefaultAutoInstDotNet
					}
				case constants.AnnotationDefaultAutoInstrumentationGo:
					if inst.Spec.Go.Image == autoInst {
						upgraded.Spec.Go.Image = u.DefaultAutoInstGo
						upgraded.Annotations[annotation] = u.DefaultAutoInstGo
					}
				case constants.AnnotationDefaultAutoInstrumentationNginx:
					if inst.Spec.Nginx.Image == autoInst {
						upgraded.Spec.Nginx.Image = u.DefaultAutoInstNginx
						upgraded.Annotations[annotation] = u.DefaultAutoInstNginx
					}
				case constants.AnnotationDefaultAutoInstrumentationPython:
					if inst.Spec.Python.Image == autoInst {
						upgraded.Spec.Python.Image = u.DefaultAutoInstPython
						upgraded.Annotations[annotation] = u.DefaultAutoInstPython
					}
				case constants.AnnotationDefaultAutoInstrumentationNodeJS:
					if inst.Spec.NodeJS.Image == autoInst {
						upgraded.Spec.NodeJS.Image = u.DefaultAutoInstNodeJS
						upgraded.Annotations[annotation] = u.DefaultAutoInstNodeJS
					}
				case constants.AnnotationDefaultAutoInstrumentationJava:
					if inst.Spec.Java.Image == autoInst {
						upgraded.Spec.Java.Image = u.DefaultAutoInstJava
						upgraded.Annotations[annotation] = u.DefaultAutoInstJava
					}
				}

			} else {
				u.Logger.V(4).Info("autoinstrumentation not enabled for this language", "flag", instCfg.id)
			}
		}
	}

	return upgraded
}
