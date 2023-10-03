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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// annotationInjectJava indicates whether java auto-instrumentation should be injected or not.
	// Possible values are "true", "false" or "<Instrumentation>" name.
	annotationInjectContainerName             = "instrumentation.opentelemetry.io/container-names"
	annotationInjectJava                      = "instrumentation.opentelemetry.io/inject-java"
	annotationInjectJavaContainersName        = "instrumentation.opentelemetry.io/java-container-names"
	annotationInjectNodeJS                    = "instrumentation.opentelemetry.io/inject-nodejs"
	annotationInjectNodeJSContainersName      = "instrumentation.opentelemetry.io/nodejs-container-names"
	annotationInjectPython                    = "instrumentation.opentelemetry.io/inject-python"
	annotationInjectPythonContainersName      = "instrumentation.opentelemetry.io/python-container-names"
	annotationInjectDotNet                    = "instrumentation.opentelemetry.io/inject-dotnet"
	annotationDotNetRuntime                   = "instrumentation.opentelemetry.io/otel-dotnet-auto-runtime"
	annotationInjectDotnetContainersName      = "instrumentation.opentelemetry.io/dotnet-container-names"
	annotationInjectGo                        = "instrumentation.opentelemetry.io/inject-go"
	annotationInjectGoContainersName          = "instrumentation.opentelemetry.io/go-container-names"
	annotationGoExecPath                      = "instrumentation.opentelemetry.io/otel-go-auto-target-exe"
	annotationInjectSdk                       = "instrumentation.opentelemetry.io/inject-sdk"
	annotationInjectSdkContainersName         = "instrumentation.opentelemetry.io/sdk-container-names"
	annotationInjectApacheHttpd               = "instrumentation.opentelemetry.io/inject-apache-httpd"
	annotationInjectApacheHttpdContainersName = "instrumentation.opentelemetry.io/apache-httpd-container-names"
	annotationInjectNginx                     = "instrumentation.opentelemetry.io/inject-nginx"
	annotationInjectNginxContainersName       = "instrumentation.opentelemetry.io/inject-nginx-container-names"
)

// annotationValue returns the effective annotationInjectJava value, based on the annotations from the pod and namespace.
func annotationValue(ns metav1.ObjectMeta, pod metav1.ObjectMeta, annotation string) string {
	// is the pod annotated with instructions to inject sidecars? is the namespace annotated?
	// if any of those is true, a sidecar might be desired.
	podAnnValue := pod.Annotations[annotation]
	nsAnnValue := ns.Annotations[annotation]

	// if the namespace value is empty, the pod annotation should be used, whatever it is
	if len(nsAnnValue) == 0 {
		return podAnnValue
	}

	// if the pod value is empty, the annotation should be used (true, false, instance)
	if len(podAnnValue) == 0 {
		return nsAnnValue
	}

	// the pod annotation isn't empty -- if it's an instance name, or false, that's the decision
	if !strings.EqualFold(podAnnValue, "true") {
		return podAnnValue
	}

	// pod annotation is 'true', and if the namespace annotation is false, we just return 'true'
	if strings.EqualFold(nsAnnValue, "false") {
		return podAnnValue
	}

	// by now, the pod annotation is 'true', and the namespace annotation is either true or an instance name
	// so, the namespace annotation can be used
	return nsAnnValue
}
