// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"context"
	"fmt"
	"maps"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

type autoInstConfig struct {
	id           string
	enabled      bool
	language     constants.InstrumentationLanguage
	defaultImage string
}

type InstrumentationUpgrade struct {
	Client                     client.Client
	Logger                     logr.Logger
	Recorder                   events.EventRecorder
	DefaultAutoInstJava        string
	DefaultAutoInstNodeJS      string
	DefaultAutoInstPython      string
	DefaultAutoInstDotNet      string
	DefaultAutoInstApacheHttpd string
	DefaultAutoInstNginx       string
	DefaultAutoInstGo          string
	defaultAnnotationToConfig  map[string]autoInstConfig
}

func NewInstrumentationUpgrade(client client.Client, logger logr.Logger, recorder events.EventRecorder, cfg config.Config) *InstrumentationUpgrade {
	defaultAnnotationToConfig := map[string]autoInstConfig{
		constants.AnnotationDefaultAutoInstrumentationApacheHttpd: {id: "enable-apache-httpd-instrumentation", enabled: cfg.EnableApacheHttpdInstrumentation, language: constants.InstrumentationLanguageApacheHttpd, defaultImage: cfg.AutoInstrumentationApacheHttpdImage},
		constants.AnnotationDefaultAutoInstrumentationDotNet:      {id: "enable-dotnet-instrumentation", enabled: cfg.EnableDotNetAutoInstrumentation, language: constants.InstrumentationLanguageDotNet, defaultImage: cfg.AutoInstrumentationDotNetImage},
		constants.AnnotationDefaultAutoInstrumentationGo:          {id: "enable-go-instrumentation", enabled: cfg.EnableGoAutoInstrumentation, language: constants.InstrumentationLanguageGo, defaultImage: cfg.AutoInstrumentationGoImage},
		constants.AnnotationDefaultAutoInstrumentationNginx:       {id: "enable-nginx-instrumentation", enabled: cfg.EnableNginxAutoInstrumentation, language: constants.InstrumentationLanguageNginx, defaultImage: cfg.AutoInstrumentationNginxImage},
		constants.AnnotationDefaultAutoInstrumentationPython:      {id: "enable-python-instrumentation", enabled: cfg.EnablePythonAutoInstrumentation, language: constants.InstrumentationLanguagePython, defaultImage: cfg.AutoInstrumentationPythonImage},
		constants.AnnotationDefaultAutoInstrumentationNodeJS:      {id: "enable-nodejs-instrumentation", enabled: cfg.EnableNodeJSAutoInstrumentation, language: constants.InstrumentationLanguageNodeJS, defaultImage: cfg.AutoInstrumentationNodeJSImage},
		constants.AnnotationDefaultAutoInstrumentationJava:        {id: "enable-java-instrumentation", enabled: cfg.EnableJavaAutoInstrumentation, language: constants.InstrumentationLanguageJava, defaultImage: cfg.AutoInstrumentationJavaImage},
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
// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations/status,verbs=get;update;patch

// ManagedInstances upgrades managed instances by the opentelemetry-operator.
func (u *InstrumentationUpgrade) ManagedInstances(ctx context.Context) error {
	u.Logger.Info("looking for managed Instrumentation instances to upgrade")
	list := &v1alpha1.InstrumentationList{}
	if err := u.Client.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		toUpgrade := list.Items[i]
		upgraded, blockedVersions := u.upgrade(ctx, toUpgrade)
		if !reflect.DeepEqual(upgraded, toUpgrade) {
			// use update instead of patch because the patch does not upgrade annotations
			if err := u.Client.Update(ctx, upgraded); err != nil {
				u.Logger.Error(err, "failed to apply changes to instance", "name", upgraded.Name, "namespace", upgraded.Namespace)
				continue
			}
		}
		// Update status if the blocked versions set has changed (including clearing it when no longer blocked).
		if !maps.Equal(upgraded.Status.UpgradeBlockedVersions, blockedVersions) {
			upgraded.Status.UpgradeBlockedVersions = blockedVersions
			if err := u.Client.Status().Update(ctx, upgraded); err != nil {
				u.Logger.Error(err, "failed to update status for blocked upgrade", "name", upgraded.Name, "namespace", upgraded.Namespace)
			}
		}
	}

	if len(list.Items) == 0 {
		u.Logger.Info("no instances to upgrade")
	}
	return nil
}

func (u *InstrumentationUpgrade) upgrade(_ context.Context, inst v1alpha1.Instrumentation) (
	upgraded *v1alpha1.Instrumentation, blockedVersions map[string]string,
) {
	upgraded = inst.DeepCopy()
	for annotation, instCfg := range u.defaultAnnotationToConfig {
		autoInst := upgraded.Annotations[annotation]
		if autoInst != "" {
			if instCfg.enabled {
				// Check if the current version is unupgradable
				if isUnupgradable, warningMsg := version.IsInstrumentationVersionUnupgradable(instCfg.language, autoInst, instCfg.defaultImage); isUnupgradable {
					msg := fmt.Sprintf("Automated upgrade blocked for %s: version is marked as unupgradable. Manual upgrade required.", instCfg.language)
					if warningMsg != "" {
						msg = fmt.Sprintf("Automated upgrade blocked for %s: %s", instCfg.language, warningMsg)
					}
					// Include the language in the event action so that two events
					// for the same Instrumentation but different languages are
					// not aggregated into one by the events.k8s.io broadcaster
					// (which keys by type/action/reason/regarding, not note).
					u.Recorder.Eventf(upgraded, nil, corev1.EventTypeWarning, "UpgradeBlocked", fmt.Sprintf("Upgrade-%s", instCfg.language), msg)
					u.Logger.Info("upgrade blocked for unupgradable instrumentation version",
						"name", inst.Name,
						"namespace", inst.Namespace,
						"language", instCfg.language,
						"image", autoInst)
					if blockedVersions == nil {
						blockedVersions = make(map[string]string)
					}
					blockedVersions[string(instCfg.language)] = msg
					continue // Skip upgrade for this language
				}

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

	return upgraded, blockedVersions
}
