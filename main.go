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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhookhandler"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	collectorupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation"
	instrumentationupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(otelv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	// registers any flags that underlying libraries might use
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	v := version.Get()

	// add flags related to this operator
	var (
		metricsAddr               string
		probeAddr                 string
		enableLeaderElection      bool
		collectorImage            string
		targetAllocatorImage      string
		autoInstrumentationJava   string
		autoInstrumentationNodeJS string
		autoInstrumentationPython string
		autoInstrumentationDotNet string
		labelsFilter              []string
		webhookPort               int
	)

	pflag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-addr", ":8081", "The address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.StringVar(&collectorImage, "collector-image", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:%s", v.OpenTelemetryCollector), "The default OpenTelemetry collector image. This image is used when no image is specified in the CustomResource.")
	pflag.StringVar(&targetAllocatorImage, "target-allocator-image", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/target-allocator:%s", v.TargetAllocator), "The default OpenTelemetry target allocator image. This image is used when no image is specified in the CustomResource.")
	pflag.StringVar(&autoInstrumentationJava, "auto-instrumentation-java-image", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:%s", v.AutoInstrumentationJava), "The default OpenTelemetry Java instrumentation image. This image is used when no image is specified in the CustomResource.")
	pflag.StringVar(&autoInstrumentationNodeJS, "auto-instrumentation-nodejs-image", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:%s", v.AutoInstrumentationNodeJS), "The default OpenTelemetry NodeJS instrumentation image. This image is used when no image is specified in the CustomResource.")
	pflag.StringVar(&autoInstrumentationPython, "auto-instrumentation-python-image", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:%s", v.AutoInstrumentationPython), "The default OpenTelemetry Python instrumentation image. This image is used when no image is specified in the CustomResource.")
	pflag.StringVar(&autoInstrumentationDotNet, "auto-instrumentation-dot-net", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-dotnet:%s", v.AutoInstrumentationDotNet), "The default OpenTelemetry DotNet instrumentation image. This image is used when no image is specified in the CustomResource.")
	pflag.StringArrayVar(&labelsFilter, "labels", []string{}, "Labels to filter away from propagating onto deploys")
	pflag.IntVar(&webhookPort, "webhook-port", 9443, "The port the webhook endpoint binds to.")
	pflag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	logger.Info("Starting the OpenTelemetry Operator",
		"opentelemetry-operator", v.Operator,
		"opentelemetry-collector", collectorImage,
		"opentelemetry-targetallocator", targetAllocatorImage,
		"auto-instrumentation-java", autoInstrumentationJava,
		"auto-instrumentation-nodejs", autoInstrumentationNodeJS,
		"auto-instrumentation-python", autoInstrumentationPython,
		"auto-instrumentation-dotnet", autoInstrumentationDotNet,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
		"labels-filter", labelsFilter,
	)

	restConfig := ctrl.GetConfigOrDie()

	// builds the operator's configuration
	ad, err := autodetect.New(restConfig)
	if err != nil {
		setupLog.Error(err, "failed to setup auto-detect routine")
		os.Exit(1)
	}

	cfg := config.New(
		config.WithLogger(ctrl.Log.WithName("config")),
		config.WithVersion(v),
		config.WithCollectorImage(collectorImage),
		config.WithTargetAllocatorImage(targetAllocatorImage),
		config.WithAutoInstrumentationJavaImage(autoInstrumentationJava),
		config.WithAutoInstrumentationNodeJSImage(autoInstrumentationNodeJS),
		config.WithAutoInstrumentationPythonImage(autoInstrumentationPython),
		config.WithAutoInstrumentationDotNetImage(autoInstrumentationDotNet),
		config.WithAutoDetect(ad),
		config.WithLabelFilters(labelsFilter),
	)

	watchNamespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if found {
		setupLog.Info("watching namespace(s)", "namespaces", watchNamespace)
	} else {
		setupLog.Info("the env var WATCH_NAMESPACE isn't set, watching all namespaces")
	}

	// see https://github.com/openshift/library-go/blob/4362aa519714a4b62b00ab8318197ba2bba51cb7/pkg/config/leaderelection/leaderelection.go#L104
	leaseDuration := time.Second * 137
	renewDeadline := time.Second * 107
	retryPeriod := time.Second * 26
	mgrOptions := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   webhookPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9f7554c3.opentelemetry.io",
		Namespace:              watchNamespace,
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		RetryPeriod:            &retryPeriod,
	}

	if strings.Contains(watchNamespace, ",") {
		mgrOptions.Namespace = ""
		mgrOptions.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(watchNamespace, ","))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	err = addDependencies(ctx, mgr, cfg, v)
	if err != nil {
		setupLog.Error(err, "failed to add/run bootstrap dependencies to the controller manager")
		os.Exit(1)
	}

	if err = controllers.NewReconciler(controllers.Params{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpenTelemetryCollector"),
		Scheme:   mgr.GetScheme(),
		Config:   cfg,
		Recorder: mgr.GetEventRecorderFor("opentelemetry-operator"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenTelemetryCollector")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&otelv1alpha1.OpenTelemetryCollector{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenTelemetryCollector")
			os.Exit(1)
		}
		if err = (&otelv1alpha1.Instrumentation{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					otelv1alpha1.AnnotationDefaultAutoInstrumentationJava:   autoInstrumentationJava,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationNodeJS: autoInstrumentationNodeJS,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationPython: autoInstrumentationPython,
					otelv1alpha1.AnnotationDefaultAutoInstrumentationDotNet: autoInstrumentationDotNet,
				},
			},
		}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Instrumentation")
			os.Exit(1)
		}

		mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
			Handler: webhookhandler.NewWebhookHandler(cfg, ctrl.Log.WithName("pod-webhook"), mgr.GetClient(),
				[]webhookhandler.PodMutator{
					sidecar.NewMutator(logger, cfg, mgr.GetClient()),
					instrumentation.NewMutator(logger, mgr.GetClient()),
				}),
		})
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addDependencies(_ context.Context, mgr ctrl.Manager, cfg config.Config, v version.Version) error {
	// run the auto-detect mechanism for the configuration
	err := mgr.Add(manager.RunnableFunc(func(_ context.Context) error {
		return cfg.StartAutoDetect()
	}))
	if err != nil {
		return fmt.Errorf("failed to start the auto-detect mechanism: %w", err)
	}
	// adds the upgrade mechanism to be executed once the manager is ready
	err = mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		up := &collectorupgrade.VersionUpgrade{
			Log:      ctrl.Log.WithName("collector-upgrade"),
			Version:  v,
			Client:   mgr.GetClient(),
			Recorder: record.NewFakeRecorder(collectorupgrade.RecordBufferSize),
		}
		return up.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade OpenTelemetryCollector instances: %w", err)
	}

	// adds the upgrade mechanism to be executed once the manager is ready
	err = mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		u := &instrumentationupgrade.InstrumentationUpgrade{
			Logger:                ctrl.Log.WithName("instrumentation-upgrade"),
			DefaultAutoInstJava:   cfg.AutoInstrumentationJavaImage(),
			DefaultAutoInstNodeJS: cfg.AutoInstrumentationNodeJSImage(),
			DefaultAutoInstPython: cfg.AutoInstrumentationPythonImage(),
			DefaultAutoInstDotNet: cfg.AutoInstrumentationDotNetImage(),
			Client:                mgr.GetClient(),
		}
		return u.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade Instrumentation instances: %w", err)
	}
	return nil
}
