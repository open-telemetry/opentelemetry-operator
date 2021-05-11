module github.com/open-telemetry/opentelemetry-operator

go 1.15

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/go-logr/logr v0.4.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/kubectl v0.21.0
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)
