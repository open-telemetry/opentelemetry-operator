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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

type SelfSignedIssuerConfig struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

type CAIssuerConfig struct {
	Name       string
	Namespace  string
	Labels     map[string]string
	SecretName string
}

func TestSelfSignedIssuer(t *testing.T) {
	taSpec := v1alpha1.TargetAllocatorSpec{}
	ta := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: taSpec,
	}

	cfg := config.New()

	expected := SelfSignedIssuerConfig{
		Name:      "my-instance-self-signed-issuer",
		Namespace: "my-namespace",
		Labels: map[string]string{
			"app.kubernetes.io/name":       "my-instance-self-signed-issuer",
			"app.kubernetes.io/instance":   "my-namespace.my-instance",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/part-of":    "opentelemetry",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
			"app.kubernetes.io/version":    "latest",
		},
	}

	params := Params{
		Config:          cfg,
		TargetAllocator: ta,
	}

	issuer := SelfSignedIssuer(params)

	assert.Equal(t, expected.Name, issuer.Name)
	assert.Equal(t, expected.Namespace, issuer.Namespace)
	assert.Equal(t, expected.Labels, issuer.Labels)
	assert.NotNil(t, issuer.Spec.SelfSigned)
}

func TestCAIssuer(t *testing.T) {
	ta := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
	}

	cfg := config.New()

	expected := CAIssuerConfig{
		Name:      "my-instance-ca-issuer",
		Namespace: "my-namespace",
		Labels: map[string]string{
			"app.kubernetes.io/name":       "my-instance-ca-issuer",
			"app.kubernetes.io/instance":   "my-namespace.my-instance",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/part-of":    "opentelemetry",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
			"app.kubernetes.io/version":    "latest",
		},
		SecretName: "my-instance-ca-cert",
	}

	params := Params{
		Config:          cfg,
		TargetAllocator: ta,
	}

	issuer := CAIssuer(params)

	assert.Equal(t, expected.Name, issuer.Name)
	assert.Equal(t, expected.Namespace, issuer.Namespace)
	assert.Equal(t, expected.Labels, issuer.Labels)
	assert.Equal(t, expected.SecretName, issuer.Spec.CA.SecretName)
}
