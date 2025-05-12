// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifests

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

// Params holds the reconciliation-specific parameters.
type Params struct {
	Client          client.Client
	Recorder        record.EventRecorder
	Scheme          *runtime.Scheme
	Log             logr.Logger
	OtelCol         v1beta1.OpenTelemetryCollector
	TargetAllocator *v1alpha1.TargetAllocator
	OpAMPBridge     v1alpha1.OpAMPBridge
	Config          config.Config
	Reviewer        rbac.SAReviewer
	ErrorAsWarning  bool
}
