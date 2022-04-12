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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

type InstrumentationUpgrade struct {
	Logger                logr.Logger
	DefaultAutoInstJava   string
	DefaultAutoInstNodeJS string
	DefaultAutoInstPython string
	DefaultAutoInstDotNet string
	Client                client.Client
}

//+kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch;update;patch

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
			if err := u.Client.Update(ctx, &upgraded); err != nil {
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

func (u *InstrumentationUpgrade) upgrade(_ context.Context, inst v1alpha1.Instrumentation) v1alpha1.Instrumentation {
	autoInstJava := inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationJava]
	if autoInstJava != "" {
		// upgrade the image only if the image matches the annotation
		if inst.Spec.Java.Image == autoInstJava {
			inst.Spec.Java.Image = u.DefaultAutoInstJava
			inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationJava] = u.DefaultAutoInstJava
		}
	}
	autoInstNodeJS := inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationNodeJS]
	if autoInstNodeJS != "" {
		// upgrade the image only if the image matches the annotation
		if inst.Spec.NodeJS.Image == autoInstNodeJS {
			inst.Spec.NodeJS.Image = u.DefaultAutoInstNodeJS
			inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationNodeJS] = u.DefaultAutoInstNodeJS
		}
	}
	autoInstPython := inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationPython]
	if autoInstPython != "" {
		// upgrade the image only if the image matches the annotation
		if inst.Spec.Python.Image == autoInstPython {
			inst.Spec.Python.Image = u.DefaultAutoInstPython
			inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationPython] = u.DefaultAutoInstPython
		}
	}
	autoInstDotnet := inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationDotNet]
	if autoInstDotnet != "" {
		// upgrade the image only if the image matches the annotation
		if inst.Spec.DotNet.Image == autoInstDotnet {
			inst.Spec.DotNet.Image = u.DefaultAutoInstDotNet
			inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationDotNet] = u.DefaultAutoInstDotNet
		}
	}
	return inst
}
