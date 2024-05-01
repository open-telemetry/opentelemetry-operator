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

package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
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
		constants.AnnotationDefaultAutoInstrumentationApacheHttpd: {constants.FlagApacheHttpd, cfg.EnableApacheHttpdAutoInstrumentation()},
		constants.AnnotationDefaultAutoInstrumentationDotNet:      {constants.FlagDotNet, cfg.EnableDotNetAutoInstrumentation()},
		constants.AnnotationDefaultAutoInstrumentationGo:          {constants.FlagGo, cfg.EnableGoAutoInstrumentation()},
		constants.AnnotationDefaultAutoInstrumentationNginx:       {constants.FlagNginx, cfg.EnableNginxAutoInstrumentation()},
		constants.AnnotationDefaultAutoInstrumentationPython:      {constants.FlagPython, cfg.EnablePythonAutoInstrumentation()},
		constants.AnnotationDefaultAutoInstrumentationNodeJS:      {constants.FlagNodeJS, cfg.EnableNodeJSAutoInstrumentation()},
		constants.AnnotationDefaultAutoInstrumentationJava:        {constants.FlagJava, cfg.EnableJavaAutoInstrumentation()},
	}

	return &InstrumentationUpgrade{
		Client:                     client,
		Logger:                     logger,
		DefaultAutoInstJava:        cfg.AutoInstrumentationJavaImage(),
		DefaultAutoInstNodeJS:      cfg.AutoInstrumentationNodeJSImage(),
		DefaultAutoInstPython:      cfg.AutoInstrumentationPythonImage(),
		DefaultAutoInstDotNet:      cfg.AutoInstrumentationDotNetImage(),
		DefaultAutoInstGo:          cfg.AutoInstrumentationGoImage(),
		DefaultAutoInstApacheHttpd: cfg.AutoInstrumentationApacheHttpdImage(),
		DefaultAutoInstNginx:       cfg.AutoInstrumentationNginxImage(),
		Recorder:                   recorder,
		defaultAnnotationToConfig:  defaultAnnotationToConfig,
	}

}

// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch;update;patch

// ManagedInstances upgrades managed instances by the opentelemetry-operator.
func (u *InstrumentationUpgrade) ManagedInstances(ctx context.Context) error {
	u.Logger.Info("looking for managed Instrumentation instances to upgrade")

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &v1alpha1.InstrumentationList{}
	if err := u.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
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
