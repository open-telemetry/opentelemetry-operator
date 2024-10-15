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

package targetallocator

import (
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// SelfSignedIssuer returns a self-signed issuer for the given instance.
func SelfSignedIssuer(params Params) *cmv1.Issuer {
	name := naming.SelfSignedIssuer(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: params.TargetAllocator.Namespace,
			Labels:    labels,
		},
		Spec: cmv1.IssuerSpec{
			IssuerConfig: cmv1.IssuerConfig{
				SelfSigned: &cmv1.SelfSignedIssuer{},
			},
		},
	}
}

// CAIssuer returns a CA issuer for the given instance.
func CAIssuer(params Params) *cmv1.Issuer {
	name := naming.CAIssuer(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: params.TargetAllocator.Namespace,
			Labels:    labels,
		},
		Spec: cmv1.IssuerSpec{
			IssuerConfig: cmv1.IssuerConfig{
				CA: &cmv1.CAIssuer{
					SecretName: naming.CACertificate(params.TargetAllocator.Name),
				},
			},
		},
	}
}
