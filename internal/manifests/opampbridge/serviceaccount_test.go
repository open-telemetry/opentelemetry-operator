// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestServiceAccountNewDefault(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	// test
	sa := ServiceAccountName(opampBridge)

	// verify
	assert.Equal(t, "my-instance-opamp-bridge", sa)
}

func TestServiceAccountOverride(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			ServiceAccount: "my-special-sa",
		},
	}

	// test
	sa := ServiceAccountName(opampBridge)

	// verify
	assert.Equal(t, "my-special-sa", sa)
}
