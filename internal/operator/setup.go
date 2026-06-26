// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	openshifttls "github.com/openshift/controller-runtime-common/pkg/tls"
	"go.uber.org/zap/zapcore"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

var setupLog = ctrl.Log.WithName("setup")

// SetupResult holds the results of common manager setup, shared between operator and webhook commands.
type SetupResult struct {
	Manager           ctrl.Manager
	Clientset         kubernetes.Interface
	Reviewer          *rbac.Reviewer
	Config            config.Config
	RestConfig        *rest.Config
	InitialTLSProfile configv1.TLSProfileSpec
	Autodetector      autodetect.AutoDetect
	TLSOpts           []func(*tls.Config)
}

// SetupManager performs the common manager setup shared between the operator and webhook commands.
// It applies configuration, creates the controller-runtime manager, sets up autodetection, and returns
// a SetupResult with all the components needed for further command-specific setup.
func SetupManager(cfg *config.Config, configFile string, opts zap.Options, scheme *k8sruntime.Scheme, leaderElection bool, startupMessage string) *SetupResult {
	applyConfigAndSetupLogger(cfg, configFile, opts)
	logger := ctrl.Log

	v := version.Get()
	logger.Info(startupMessage,
		"opentelemetry-operator", v.Operator,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
		"config", cfg.ToStringMap(),
	)

	restConfig := ctrl.GetConfigOrDie()
	namespaces := parseWatchNamespaces(cfg.WatchNamespace)
	tlsOptsFuncs, initialTLSProfile := buildTLSOptions(*cfg, restConfig, scheme)

	if cfg.TLS.ConfigureOperands {
		tlsCfg := &tls.Config{}
		for _, t := range tlsOptsFuncs {
			t(tlsCfg)
		}
		cfg.Internal.OperandTLSProfile = components.NewStaticTLSProfile(tlsCfg.MinVersion, tlsCfg.CipherSuites)
	}

	metricsOptions := buildMetricsOptions(*cfg, tlsOptsFuncs)

	mgrOptions := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsOptions,
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         leaderElection,
		PprofBindAddress:       cfg.PprofAddr,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    cfg.WebhookPort,
			TLSOpts: tlsOptsFuncs,
		}),
		Cache: cache.Options{
			DefaultNamespaces: namespaces,
		},
	}

	if leaderElection {
		leaseDuration := time.Second * 137
		renewDeadline := time.Second * 107
		retryPeriod := time.Second * 26
		mgrOptions.LeaderElectionID = "9f7554c3.opentelemetry.io"
		mgrOptions.LeaderElectionReleaseOnCancel = true
		mgrOptions.LeaseDuration = &leaseDuration
		mgrOptions.RenewDeadline = &renewDeadline
		mgrOptions.RetryPeriod = &retryPeriod
	}

	mgr, err := ctrl.NewManager(restConfig, mgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes clientset")
	}

	reviewer := rbac.NewReviewer(clientset)

	configLog := ctrl.Log.WithName("config")
	ad, err := autodetect.New(restConfig, reviewer)
	if err != nil {
		setupLog.Error(err, "failed to setup auto-detect routine")
		os.Exit(1)
	}

	if err = autodetect.ApplyAutoDetect(ad, cfg, configLog); err != nil {
		setupLog.Error(err, "failed to autodetect config variables")
	}

	setupLog.Info("Native sidecar", "enabled", cfg.Internal.NativeSidecarSupport)

	validateFilterPatterns(*cfg)

	return &SetupResult{
		Manager:           mgr,
		Clientset:         clientset,
		Reviewer:          reviewer,
		Config:            *cfg,
		RestConfig:        restConfig,
		InitialTLSProfile: initialTLSProfile,
		Autodetector:      ad,
		TLSOpts:           tlsOptsFuncs,
	}
}

// SetupTLSProfileWatcher configures the TLS security profile watcher for OpenShift clusters.
// When the cluster's TLS profile changes, it cancels the context to trigger a graceful restart.
func SetupTLSProfileWatcher(mgr ctrl.Manager, initialTLSProfile configv1.TLSProfileSpec, cancel context.CancelFunc) {
	watcher := &openshifttls.SecurityProfileWatcher{
		Client:                mgr.GetClient(),
		InitialTLSProfileSpec: initialTLSProfile,
		OnProfileChange: func(_ context.Context, _, _ configv1.TLSProfileSpec) {
			setupLog.Info("TLS security profile changed, triggering graceful restart")
			cancel()
		},
	}
	if err := watcher.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup TLS profile watcher")
		os.Exit(1)
	}
}

// AddHealthChecks registers health and readiness probes on the manager.
// If includeWebhook is true, it also adds a readiness check for the webhook server.
func AddHealthChecks(mgr ctrl.Manager, includeWebhook bool) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	if includeWebhook {
		if err := mgr.AddReadyzCheck("webhook", mgr.GetWebhookServer().StartedChecker()); err != nil {
			setupLog.Error(err, "unable to set up webhook ready check")
			os.Exit(1)
		}
	}
}

// StartManager starts the controller-runtime manager and exits on error.
func StartManager(ctx context.Context, mgr ctrl.Manager) {
	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func applyConfigAndSetupLogger(cfg *config.Config, configFile string, opts zap.Options) {
	err := cfg.Apply(configFile)
	if err != nil {
		fmt.Printf("configuration error: %v\n", err)
		os.Exit(1)
	}

	opts.EncoderConfigOptions = append(opts.EncoderConfigOptions, func(ec *zapcore.EncoderConfig) {
		ec.MessageKey = cfg.Zap.MessageKey
		ec.LevelKey = cfg.Zap.LevelKey
		ec.TimeKey = cfg.Zap.TimeKey
		if cfg.Zap.LevelFormat == "lowercase" {
			ec.EncodeLevel = zapcore.LowercaseLevelEncoder
		} else {
			ec.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	})

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)
}

func parseWatchNamespaces(watchNamespace string) map[string]cache.Config {
	if watchNamespace != "" {
		setupLog.Info("watching namespace(s)", "namespaces", watchNamespace)
		namespaces := map[string]cache.Config{}
		for ns := range strings.SplitSeq(watchNamespace, ",") {
			namespaces[ns] = cache.Config{}
		}
		return namespaces
	}
	setupLog.Info("watching all namespaces")
	return nil
}

func buildTLSOptions(cfg config.Config, restCfg *rest.Config, scheme *k8sruntime.Scheme) ([]func(*tls.Config), configv1.TLSProfileSpec) {
	optionsTlSOptsFuncs := []func(*tls.Config){
		func(c *tls.Config) {
			if tlsErr := cfg.TLS.ApplyTLSConfig(c); tlsErr != nil {
				setupLog.Error(tlsErr, "error setting up TLS")
			}
		},
	}

	var initialTLSProfileSpec configv1.TLSProfileSpec
	if cfg.TLS.UseClusterProfile {
		tempClient, errClient := client.New(restCfg, client.Options{Scheme: scheme})
		if errClient != nil {
			setupLog.Error(errClient, "unable to create temporary client for TLS profile fetch")
			os.Exit(1)
		}

		var err error
		initialTLSProfileSpec, err = openshifttls.FetchAPIServerTLSProfile(context.Background(), tempClient)
		if err != nil {
			setupLog.Error(err, "unable to get TLS profile from cluster")
			os.Exit(1)
		}

		tlsConfigFunc, unsupportedCiphers := openshifttls.NewTLSConfigFromProfile(initialTLSProfileSpec)
		if len(unsupportedCiphers) > 0 {
			setupLog.Info("some TLS ciphers from cluster profile are not supported by Go", "unsupportedCiphers", unsupportedCiphers)
		}

		optionsTlSOptsFuncs = append(optionsTlSOptsFuncs, tlsConfigFunc)
	}

	return optionsTlSOptsFuncs, initialTLSProfileSpec
}

func buildMetricsOptions(cfg config.Config, tlsOpts []func(*tls.Config)) metricsserver.Options {
	metricsOptions := metricsserver.Options{
		BindAddress: cfg.MetricsAddr,
	}
	if cfg.MetricsSecure {
		metricsOptions.SecureServing = true
		metricsOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
		metricsOptions.TLSOpts = tlsOpts
		if cfg.MetricsTLSCertFile != "" && cfg.MetricsTLSKeyFile != "" {
			metricsOptions.CertDir = filepath.Dir(cfg.MetricsTLSCertFile)
			metricsOptions.CertName = filepath.Base(cfg.MetricsTLSCertFile)
			metricsOptions.KeyName = filepath.Base(cfg.MetricsTLSKeyFile)
		}
	}
	return metricsOptions
}

func validateFilterPatterns(cfg config.Config) {
	if cfg.AnnotationsFilter != nil {
		for _, basePattern := range cfg.AnnotationsFilter {
			_, compileErr := regexp.Compile(basePattern)
			if compileErr != nil {
				setupLog.Error(compileErr, "could not compile the regexp pattern for Annotations filter")
			}
		}
	}
	if cfg.LabelsFilter != nil {
		for _, basePattern := range cfg.LabelsFilter {
			_, compileErr := regexp.Compile(basePattern)
			if compileErr != nil {
				setupLog.Error(compileErr, "could not compile the regexp pattern for Labels filter")
			}
		}
	}
}
