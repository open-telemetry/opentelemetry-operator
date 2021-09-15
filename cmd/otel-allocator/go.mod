module github.com/otel-allocator

go 1.17

require (
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-kit/log v0.1.0
	github.com/go-logr/logr v0.4.0
	github.com/gorilla/mux v1.8.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/prometheus v1.8.2-0.20210621150501-ff58416a0b02
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/component-base v0.22.1 // indirect
	sigs.k8s.io/controller-runtime v0.9.6
)
