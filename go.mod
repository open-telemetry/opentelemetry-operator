module github.com/open-telemetry/opentelemetry-operator

go 1.13

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0 // keep the Makefile in sync!
	github.com/onsi/gomega v1.8.1
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	k8s.io/kubectl v0.18.6
	sigs.k8s.io/controller-runtime v0.6.0
)
