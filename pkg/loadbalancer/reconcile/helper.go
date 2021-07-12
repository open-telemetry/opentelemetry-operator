package reconcile

import (
	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	lbadapters "github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer/adapters"
)

func checkMode(mode v1alpha1.Mode, lbMode v1alpha1.LbMode) bool {
	deploy := false
	if mode == v1alpha1.ModeStatefulSet && len(lbMode) > 0 {
		deploy = true
	}
	return deploy
}

func checkConfig(params Params) (map[interface{}]interface{}, error) {
	config, err := adapters.ConfigFromString(params.Instance.Spec.Config)
	if err != nil {
		return nil, err
	}

	promConfig, err := lbadapters.ConfigToPromConfig(config)
	if err != nil {
		return nil, err
	}

	return promConfig, nil
}
