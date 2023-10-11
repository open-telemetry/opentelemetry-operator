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
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

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
	autoInstJava := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationJava]
	if autoInstJava != "" {
		if featuregate.EnableJavaAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.Java.Image == autoInstJava {
				upgraded.Spec.Java.Image = u.DefaultAutoInstJava
			}
		} else {
			u.Logger.Error(nil, "support for Java auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for Java auto instrumentation is not enabled")
		}
	}
	autoInstNodeJS := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationNodeJS]
	if autoInstNodeJS != "" {
		if featuregate.EnableNodeJSAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.NodeJS.Image == autoInstNodeJS {
				upgraded.Spec.NodeJS.Image = u.DefaultAutoInstNodeJS
			}
		} else {
			u.Logger.Error(nil, "support for NodeJS auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for NodeJS auto instrumentation is not enabled")
		}
	}
	autoInstPython := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationPython]
	if autoInstPython != "" {
		if featuregate.EnablePythonAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.Python.Image == autoInstPython {
				upgraded.Spec.Python.Image = u.DefaultAutoInstPython
			}
		} else {
			u.Logger.Error(nil, "support for Python auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for Python auto instrumentation is not enabled")
		}
	}
	autoInstDotnet := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationDotNet]
	if autoInstDotnet != "" {
		if featuregate.EnableDotnetAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.DotNet.Image == autoInstDotnet {
				upgraded.Spec.DotNet.Image = u.DefaultAutoInstDotNet
			}
		} else {
			u.Logger.Error(nil, "support for .NET auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for .NET auto instrumentation is not enabled")
		}
	}
	autoInstGo := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationGo]
	if autoInstGo != "" {
		if featuregate.EnableGoAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.Go.Image == autoInstGo {
				upgraded.Spec.Go.Image = u.DefaultAutoInstGo
			}
		} else {
			u.Logger.Error(nil, "support for Go auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for Go auto instrumentation is not enabled")
		}
	}
	autoInstApacheHttpd := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationApacheHttpd]
	if autoInstApacheHttpd != "" {
		if featuregate.EnableApacheHTTPAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.ApacheHttpd.Image == autoInstApacheHttpd {
				upgraded.Spec.ApacheHttpd.Image = u.DefaultAutoInstApacheHttpd
			}
		} else {
			u.Logger.Error(nil, "support for Apache HTTPD auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for Apache HTTPD auto instrumentation is not enabled")
		}
	}
	autoInstNginx := upgraded.Annotations[constants.AnnotationDefaultAutoInstrumentationNginx]
	if autoInstNginx != "" {
		if featuregate.EnableNginxAutoInstrumentationSupport.IsEnabled() {
			// upgrade the image only if the image matches the annotation
			if inst.Spec.Nginx.Image == autoInstNginx {
				upgraded.Spec.Nginx.Image = u.DefaultAutoInstNginx
			}
		} else {
			u.Logger.Error(nil, "support for Nginx auto instrumentation is not enabled")
			u.Recorder.Event(upgraded, "Warning", "InstrumentationUpgradeRejected", "support for Nginx auto instrumentation is not enabled")
		}
	}
	return upgraded
}
