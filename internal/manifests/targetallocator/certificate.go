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
	"fmt"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// / CACertificate returns a CA Certificate for the given instance.
func CACertificate(params Params) *cmv1.Certificate {
	name := naming.CACertificate(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: cmv1.CertificateSpec{
			IsCA:       true,
			CommonName: naming.CACertificate(params.TargetAllocator.Name),
			Subject: &cmv1.X509Subject{
				OrganizationalUnits: []string{"opentelemetry-operator"},
			},
			SecretName: naming.CACertificate(params.TargetAllocator.Name),
			IssuerRef: cmmeta.ObjectReference{
				Name: naming.SelfSignedIssuer(params.TargetAllocator.Name),
				Kind: "Issuer",
			},
		},
	}
}

// ServingCertificate returns a serving Certificate for the given instance.
func ServingCertificate(params Params) *cmv1.Certificate {
	name := naming.TAServerCertificate(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: cmv1.CertificateSpec{
			DNSNames: []string{
				naming.TAService(params.TargetAllocator.Name),
				fmt.Sprintf("%s.%s.svc", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
			},
			IssuerRef: cmmeta.ObjectReference{
				Kind: "Issuer",
				Name: naming.CAIssuer(params.TargetAllocator.Name),
			},
			Usages: []cmv1.KeyUsage{
				cmv1.UsageClientAuth,
				cmv1.UsageServerAuth,
			},
			SecretName: naming.TAServerCertificate(params.TargetAllocator.Name),
			Subject: &cmv1.X509Subject{
				OrganizationalUnits: []string{"opentelemetry-operator"},
			},
		},
	}
}

// ClientCertificate returns a client Certificate for the given instance.
func ClientCertificate(params Params) *cmv1.Certificate {
	name := naming.TAClientCertificate(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: cmv1.CertificateSpec{
			DNSNames: []string{
				naming.TAService(params.TargetAllocator.Name),
				fmt.Sprintf("%s.%s.svc", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
			},
			IssuerRef: cmmeta.ObjectReference{
				Kind: "Issuer",
				Name: naming.CAIssuer(params.TargetAllocator.Name),
			},
			Usages: []cmv1.KeyUsage{
				cmv1.UsageClientAuth,
				cmv1.UsageServerAuth,
			},
			SecretName: naming.TAClientCertificate(params.TargetAllocator.Name),
			Subject: &cmv1.X509Subject{
				OrganizationalUnits: []string{"opentelemetry-operator"},
			},
		},
	}
}
