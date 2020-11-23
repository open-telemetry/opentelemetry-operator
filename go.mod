module github.com/open-telemetry/opentelemetry-operator

go 1.13

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/go-logr/logr v0.3.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/kubectl v0.19.4
	sigs.k8s.io/controller-runtime v0.6.4
)

replace (
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)
