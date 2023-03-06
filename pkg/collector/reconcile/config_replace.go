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

package reconcile

import (
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/featuregate"

	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/http"
	_ "github.com/prometheus/prometheus/discovery/install" // Package install has the side-effect of registering all builtin.
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

var (
	// EnableTargetAllocatorRewrite is the feature gate that controls whether the collector's configuration should
	// automatically be rewritten when the target allocator is enabled.
	EnableTargetAllocatorRewrite = featuregate.GlobalRegistry().MustRegister(
		"operator.enableTargetAllocatorRewrite",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("controls whether the operator should configure the collector's targetAllocator configuration"))
)

type targetAllocator struct {
	Endpoint    string        `yaml:"endpoint"`
	Interval    time.Duration `yaml:"interval"`
	CollectorID string        `yaml:"collector_id"`
}

type Config struct {
	PromConfig        *promconfig.Config `yaml:"config"`
	TargetAllocConfig *targetAllocator   `yaml:"target_allocator,omitempty"`
}

func ReplaceConfig(instance v1alpha1.OpenTelemetryCollector) (string, error) {
	if !instance.Spec.TargetAllocator.Enabled {
		return instance.Spec.Config, nil
	}
	config, getStringErr := adapters.ConfigFromString(instance.Spec.Config)
	if getStringErr != nil {
		return "", getStringErr
	}

	promCfgMap, getCfgPromErr := ta.ConfigToPromConfig(instance.Spec.Config)
	if getCfgPromErr != nil {
		return "", getCfgPromErr
	}

	// yaml marshaling/unsmarshaling is preferred because of the problems associated with the conversion of map to a struct using mapstructure
	promCfg, marshalErr := yaml.Marshal(promCfgMap)
	if marshalErr != nil {
		return "", marshalErr
	}

	var cfg Config
	if marshalErr = yaml.UnmarshalStrict(promCfg, &cfg); marshalErr != nil {
		return "", fmt.Errorf("error unmarshaling YAML: %w", marshalErr)
	}

	for i := range cfg.PromConfig.ScrapeConfigs {
		escapedJob := url.QueryEscape(cfg.PromConfig.ScrapeConfigs[i].JobName)
		cfg.PromConfig.ScrapeConfigs[i].ServiceDiscoveryConfigs = discovery.Configs{
			&http.SDConfig{
				URL: fmt.Sprintf("http://%s:80/jobs/%s/targets?collector_id=$POD_NAME", naming.TAService(instance), escapedJob),
			},
		}
	}

	if EnableTargetAllocatorRewrite.IsEnabled() {
		cfg.TargetAllocConfig = &targetAllocator{
			Endpoint:    fmt.Sprintf("http://%s:80", naming.TAService(instance)),
			Interval:    30 * time.Second,
			CollectorID: "${POD_NAME}",
		}
	}

	// type coercion checks are handled in the ConfigToPromConfig method above
	config["receivers"].(map[interface{}]interface{})["prometheus"] = cfg

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
