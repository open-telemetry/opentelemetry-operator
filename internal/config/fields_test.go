// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/operator-framework/operator-lib/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configTriple struct {
	yaml string
	flag string
	env  string
}

// expectedTriples is the single-source-of-truth contract: every configurable field maps
// to exactly these (config-file key, CLI flag, environment variable) names. It fails
// loudly if a field is added without tags, a name drifts, or the three mechanisms stop
// agreeing. Names are intentionally not uniform (e.g. the dotnet/service-monitor entries);
// reconciling them is a separate, breaking change.
var expectedTriples = []configTriple{
	{"targetallocator-image", "target-allocator-image", "RELATED_IMAGE_TARGET_ALLOCATOR"},
	{"operatoropampbridge-image", "operator-opamp-bridge-image", "RELATED_IMAGE_OPERATOR_OPAMP_BRIDGE"},
	{"auto-instrumentation-ruby-image", "auto-instrumentation-ruby-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_RUBY"},
	{"auto-instrumentation-python-image", "auto-instrumentation-python-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_PYTHON"},
	{"collector-image", "collector-image", "RELATED_IMAGE_COLLECTOR"},
	{"clusterobservability-collector-image", "clusterobservability-collector-image", ""},
	{"collector-configmap-entry", "", ""},
	{"enable-multi-instrumentation", "enable-multi-instrumentation", "ENABLE_MULTI_INSTRUMENTATION"},
	{"enable-apache-httpd-instrumentation", "enable-apache-httpd-instrumentation", "ENABLE_APACHE_HTTPD_AUTO_INSTRUMENTATION"},
	{"enable-php-auto-instrumentation", "enable-php-instrumentation", "ENABLE_PHP_AUTO_INSTRUMENTATION"},
	{"enable-dot-net-auto-instrumentation", "enable-dotnet-instrumentation", "ENABLE_DOTNET_AUTO_INSTRUMENTATION"},
	{"enable-go-auto-instrumentation", "enable-go-instrumentation", "ENABLE_GO_AUTO_INSTRUMENTATION"},
	{"enable-nginx-auto-instrumentation", "enable-nginx-instrumentation", "ENABLE_NGINX_AUTO_INSTRUMENTATION"},
	{"enable-ruby-auto-instrumentation", "enable-ruby-instrumentation", "ENABLE_RUBY_AUTO_INSTRUMENTATION"},
	{"enable-python-auto-instrumentation", "enable-python-instrumentation", "ENABLE_PYTHON_AUTO_INSTRUMENTATION"},
	{"enable-node-js-auto-instrumentation", "enable-nodejs-instrumentation", "ENABLE_NODEJS_AUTO_INSTRUMENTATION"},
	{"enable-java-auto-instrumentation", "enable-java-instrumentation", "ENABLE_JAVA_AUTO_INSTRUMENTATION"},
	{"auto-instrumentation-php-image", "auto-instrumentation-php-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_PHP"},
	{"auto-instrumentation-dot-net-image", "auto-instrumentation-dotnet-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_DOTNET"},
	{"auto-instrumentation-go-image", "auto-instrumentation-go-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_GO"},
	{"auto-instrumentation-apache-httpd-image", "auto-instrumentation-apache-httpd-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_APACHE_HTTPD"},
	{"auto-instrumentation-nginx-image", "auto-instrumentation-nginx-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_NGINX"},
	{"target-allocator-configmap-entry", "", ""},
	{"operator-op-amp-bridge-configmap-entry", "", ""},
	{"auto-instrumentation-node-js-image", "auto-instrumentation-nodejs-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_NODEJS"},
	{"auto-instrumentation-java-image", "auto-instrumentation-java-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_JAVA"},
	{"openshift-create-dashboard", "openshift-create-dashboard", "OPENSHIFT_CREATE_DASHBOARD"},
	{"open-shift-routes-availability", "", ""},
	{"gateway-apis-availability", "", ""},
	{"prometheus-cr-availability", "", ""},
	{"cert-manager-availability", "", ""},
	{"target-allocator-availability", "", ""},
	{"collector-availability", "", ""},
	{"opampbridge-availability", "", ""},
	{"ignore-missing-collector-crds", "ignore-missing-collector-crds", "IGNORE_MISSING_COLLECTOR_CRDS"},
	{"labels-filter", "labels-filter", "LABELS_FILTER"},
	{"annotations-filter", "annotations-filter", "ANNOTATIONS_FILTER"},
	{"metrics-addr", "metrics-addr", "METRICS_ADDR"},
	{"metrics-secure", "metrics-secure", "METRICS_SECURE"},
	{"metrics-tls-cert-file", "metrics-tls-cert-file", "METRICS_TLS_CERT_FILE"},
	{"metrics-tls-key-file", "metrics-tls-key-file", "METRICS_TLS_KEY_FILE"},
	{"health-probe-addr", "health-probe-addr", "HEALTH_PROBE_ADDR"},
	{"pprof-addr", "pprof-addr", "PPROF_ADDR"},
	{"enable-leader-election", "enable-leader-election", "ENABLE_LEADER_ELECTION"},
	{"enable-cr-metrics", "enable-cr-metrics", "ENABLE_CR_METRICS"},
	{"create-service-monitor-operator-metrics", "create-sm-operator-metrics", "CREATE_SM_OPERATOR_METRICS"},
	{"webhook-port", "webhook-port", "WEBHOOK_PORT"},
	{"fips-disabled-components", "fips-disabled-components", "FIPS_DISABLED_COMPONENTS"},
	{"watch-namespace", "watch-namespace", "WATCH_NAMESPACE"},
	{"tls.useclusterprofile", "tls-cluster-profile", "TLS_CLUSTER_PROFILE"},
	{"tls.configureoperands", "tls-configure-operands", "TLS_CONFIGURE_OPERANDS"},
	{"tls.minversion", "tls-min-version", "TLS_MIN_VERSION"},
	{"tls.ciphersuites", "tls-cipher-suites", "TLS_CIPHER_SUITES"},
	{"zap.message-key", "zap-message-key", "ZAP_MESSAGE_KEY"},
	{"zap.time-key", "zap-time-key", "ZAP_TIME_KEY"},
	{"zap.level-key", "zap-level-key", "ZAP_LEVEL_KEY"},
	{"zap.level-format", "zap-level-format", "ZAP_LEVEL_FORMAT"},
	{"enable-webhooks", "enable-webhooks", "ENABLE_WEBHOOKS"},
	{"feature-gates", "feature-gates", "FEATURE_GATES"},
	{"instrumentations", "", ""},
	{"enable-instrumentation-crds", "", ""},
	{"openshift-webhook-replicas", "openshift-webhook-replicas", "OPENSHIFT_WEBHOOK_REPLICAS"},
}

func TestConfigFieldTriples(t *testing.T) {
	var got []configTriple
	walkConfig(reflect.ValueOf(New()), "", func(fs fieldSpec) {
		if fs.yaml == "" && fs.flag == "" && fs.env == "" {
			return
		}
		got = append(got, configTriple{fs.yaml, fs.flag, fs.env})
	})
	assert.ElementsMatch(t, expectedTriples, got,
		"config field tags drifted: update the struct tags or expectedTriples together")
}

// TestApplyCLIDefaultsUnchanged proves the generic flag readback does not clobber defaults
// when no flags are passed, i.e. registration defaults round-trip exactly to New().
func TestApplyCLIDefaultsUnchanged(t *testing.T) {
	oldArgs := args
	args = nil
	t.Cleanup(func() { args = oldArgs })

	c := New()
	require.NoError(t, ApplyCLI(&c))
	require.Equal(t, New(), c)
}

// TestApplyEnvDefaultsUnchanged proves an empty environment leaves the config at its
// defaults (ProxyEnvVars aside, which is always read from the environment).
func TestApplyEnvDefaultsUnchanged(t *testing.T) {
	c := New()
	ApplyEnvVars(&c)

	expected := New()
	expected.ProxyEnvVars = proxy.ReadProxyVarsFromEnv()
	require.Equal(t, expected, c)
}

// TestPrecedence confirms CLI beats env beats file beats default across the field kinds.
func TestPrecedence(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte(
		"metrics-addr: from-file\n"+
			"webhook-port: 1111\n"+
			"openshift-webhook-replicas: 1\n"+
			"enable-webhooks: false\n",
	), 0o644))

	t.Run("file beats default", func(t *testing.T) {
		oldArgs := args
		args = nil
		t.Cleanup(func() { args = oldArgs })

		c := New()
		require.NoError(t, c.Apply(file))
		require.Equal(t, "from-file", c.MetricsAddr)
		require.Equal(t, 1111, c.WebhookPort)
		require.Equal(t, int32(1), c.OpenShiftWebhookReplicas)
		require.False(t, c.EnableWebhooks)
	})

	t.Run("env beats file", func(t *testing.T) {
		oldArgs := args
		args = nil
		t.Cleanup(func() { args = oldArgs })
		t.Setenv("METRICS_ADDR", "from-env")
		t.Setenv("WEBHOOK_PORT", "2222")

		c := New()
		require.NoError(t, c.Apply(file))
		require.Equal(t, "from-env", c.MetricsAddr)
		require.Equal(t, 2222, c.WebhookPort)
	})

	t.Run("cli beats env and file", func(t *testing.T) {
		oldArgs := args
		args = []string{"--metrics-addr=from-cli", "--webhook-port=3333"}
		t.Cleanup(func() { args = oldArgs })
		t.Setenv("METRICS_ADDR", "from-env")
		t.Setenv("WEBHOOK_PORT", "2222")

		c := New()
		require.NoError(t, c.Apply(file))
		require.Equal(t, "from-cli", c.MetricsAddr)
		require.Equal(t, 3333, c.WebhookPort)
	})
}

// TestSliceFlagTypes locks the StringArray (no comma split) vs StringSlice (comma split)
// difference for the filter flags and cipher suites, and the always-split env behavior.
func TestSliceFlagTypes(t *testing.T) {
	t.Run("cli", func(t *testing.T) {
		oldArgs := args
		args = []string{"--labels-filter=a,b", "--tls-cipher-suites=c,d"}
		t.Cleanup(func() { args = oldArgs })

		c := New()
		require.NoError(t, ApplyCLI(&c))
		require.Equal(t, []string{"a,b"}, c.LabelsFilter, "StringArray must not comma-split")
		require.Equal(t, []string{"c", "d"}, c.TLS.CipherSuites, "StringSlice must comma-split")
	})

	t.Run("env", func(t *testing.T) {
		t.Setenv("LABELS_FILTER", "a,b")
		t.Setenv("TLS_CIPHER_SUITES", "c,d")

		c := New()
		ApplyEnvVars(&c)
		require.Equal(t, []string{"a", "b"}, c.LabelsFilter)
		require.Equal(t, []string{"c", "d"}, c.TLS.CipherSuites)
	})
}

// forEachLeafField recursively visits every leaf field of t, descending into the nested
// config structs exactly as the runtime walker does, so embedded TLS/Zap fields are
// covered too.
func forEachLeafField(t reflect.Type, visit func(reflect.StructField)) {
	for _, sf := range reflect.VisibleFields(t) {
		if isNestedConfig(sf.Type) {
			forEachLeafField(sf.Type, visit)
			continue
		}
		visit(sf)
	}
}

// TestEveryFieldDeclaresConfigTags enforces that every field makes an explicit choice about
// each configuration mechanism: it must carry a flag and an env tag (a real name or "-"),
// and any field that is actually configurable must also document itself with usage and group.
// This is the guard requested in the PR review so a newly added field cannot silently lack a
// binding. See issue: https://github.com/open-telemetry/opentelemetry-operator/issues/3565.
func TestEveryFieldDeclaresConfigTags(t *testing.T) {
	forEachLeafField(reflect.TypeFor[Config](), func(sf reflect.StructField) {
		flag, hasFlag := sf.Tag.Lookup("flag")
		env, hasEnv := sf.Tag.Lookup("env")

		require.Truef(t, hasFlag, "field %s is missing a flag tag (use flag:\"-\" when it has no CLI flag)", sf.Name)
		require.Truef(t, hasEnv, "field %s is missing an env tag (use env:\"-\" when it has no environment variable)", sf.Name)

		configurable := (flag != "" && flag != "-") || (env != "" && env != "-")
		if configurable {
			require.NotEmptyf(t, sf.Tag.Get("usage"), "configurable field %s must have a usage tag", sf.Name)
			require.NotEmptyf(t, sf.Tag.Get("group"), "configurable field %s must have a group tag", sf.Name)
		}
	})
}

// fieldSnapshot returns the configurable fields of c keyed by their yaml path, so two
// configs can be diffed to find which fields a single flag or env var changed.
func fieldSnapshot(c Config) map[string]any {
	out := map[string]any{}
	walkConfig(reflect.ValueOf(c), "", func(fs fieldSpec) {
		if fs.yaml == "" {
			return // runtime-only fields (Internal, ProxyEnvVars) have no stable key
		}
		out[fs.yaml] = fs.value.Interface()
	})
	return out
}

// changedFields returns the yaml keys whose values differ between two configs.
func changedFields(a, b Config) []string {
	sa, sb := fieldSnapshot(a), fieldSnapshot(b)
	var changed []string
	for k, va := range sa {
		if !reflect.DeepEqual(va, sb[k]) {
			changed = append(changed, k)
		}
	}
	return changed
}

// sentinelArg returns a string value guaranteed to differ from the field's default, for
// the field's kind, plus whether the kind is supported.
func sentinelArg(fs fieldSpec) (string, bool) {
	switch fs.value.Kind() {
	case reflect.String:
		if fs.value.String() == "zzsentinel" {
			return "zzsentinel2", true
		}
		return "zzsentinel", true
	case reflect.Bool:
		return strconv.FormatBool(!fs.value.Bool()), true
	case reflect.Int, reflect.Int32:
		return strconv.FormatInt(fs.value.Int()+1, 10), true
	case reflect.Slice:
		if _, ok := reflect.TypeAssert[[]string](fs.value); ok {
			return "zzsentinel", true
		}
		return "", false
	default:
		return "", false
	}
}

// TestEachFlagBindsToItsOwnField sets every generically-registered flag, one at a time, and
// asserts it changes exactly its own field and nothing else. This is the before/after
// binding coverage requested in the PR review: it proves each flag is wired to the right
// field, catching any cross-wiring the old hand-written switch could have hidden.
func TestEachFlagBindsToItsOwnField(t *testing.T) {
	type target struct {
		key, flag, value string
	}
	var targets []target
	walkConfig(reflect.ValueOf(New()), "", func(fs fieldSpec) {
		if fs.flag == "" || fs.manual {
			return
		}
		v, ok := sentinelArg(fs)
		require.Truef(t, ok, "unsupported kind for flag %s", fs.flag)
		targets = append(targets, target{fs.yaml, fs.flag, v})
	})
	require.NotEmpty(t, targets)

	for _, tg := range targets {
		t.Run(tg.flag, func(t *testing.T) {
			oldArgs := args
			args = []string{"--" + tg.flag + "=" + tg.value}
			t.Cleanup(func() { args = oldArgs })

			c := New()
			require.NoError(t, ApplyCLI(&c))
			require.Equal(t, []string{tg.key}, changedFields(New(), c),
				"flag --%s must change only the %q field", tg.flag, tg.key)
		})
	}
}

// TestEachEnvBindsToItsOwnField is the environment-variable counterpart of
// TestEachFlagBindsToItsOwnField.
func TestEachEnvBindsToItsOwnField(t *testing.T) {
	type target struct {
		key, env, value string
	}
	var targets []target
	walkConfig(reflect.ValueOf(New()), "", func(fs fieldSpec) {
		if fs.env == "" {
			return
		}
		v, ok := sentinelArg(fs)
		require.Truef(t, ok, "unsupported kind for env %s", fs.env)
		targets = append(targets, target{fs.yaml, fs.env, v})
	})
	require.NotEmpty(t, targets)

	for _, tg := range targets {
		t.Run(tg.env, func(t *testing.T) {
			t.Setenv(tg.env, tg.value)

			c := New()
			ApplyEnvVars(&c)
			require.Equal(t, []string{tg.key}, changedFields(New(), c),
				"env %s must change only the %q field", tg.env, tg.key)
		})
	}
}
