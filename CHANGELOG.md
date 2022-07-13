Changes by Version
==================

0.55.0
-------------------
### 🧰 Bug fixes 🧰
* Fixing monitor configuration ([#966](https://github.com/open-telemetry/opentelemetry-operator/pull/966), [@yuriolisa](https://github.com/yuriolisa))
* Fix Pod Mutation loop ([#953](https://github.com/open-telemetry/opentelemetry-operator/pull/953), [@mat-rumian](https://github.com/mat-rumian))
* Fix the issue that the number of target-allocator replicas  ([#951](https://github.com/open-telemetry/opentelemetry-operator/pull/951), [@CoderPoet](https://github.com/CoderPoet))
### 💡 Enhancements 💡
* Update Python auto-instrumentation  0.32b0 ([#961](https://github.com/open-telemetry/opentelemetry-operator/pull/961), [@mat-rumian](https://github.com/mat-rumian))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.55.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.55.0)
* [OpenTelemetry Contrib - v0.55.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.55.0)

0.54.0
-------------------
### 🧰 Bug fixes 🧰
* Fix parameter encoding issue ([#930](https://github.com/open-telemetry/opentelemetry-operator/pull/930), [@jaronoff97](https://github.com/jaronoff97))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.54.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.54.0)
* [OpenTelemetry Contrib - v0.54.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.54.0)

0.53.0
-------------------
### 💡 Enhancements 💡
* Print TA pod logs in e2e smoke test ([#920](https://github.com/open-telemetry/opentelemetry-operator/pull/920), [@pavolloffay](https://github.com/pavolloffay))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.53.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.53.0)
* [OpenTelemetry Contrib - v0.53.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.53.0)

0.52.0
-------------------
### 🚀 New components 🚀
* Add creation of ServiceAccount to the Target Allocator ([#836](https://github.com/open-telemetry/opentelemetry-operator/pull/836), [@jaronoff97](https://github.com/jaronoff97))
* Cross namespace instrumentation ([#889](https://github.com/open-telemetry/opentelemetry-operator/pull/889), [@tKe](https://github.com/tKe))
* Added extra cli flag webhook-port ([#899](https://github.com/open-telemetry/opentelemetry-operator/pull/899), [@abelperezok](https://github.com/abelperezok))
### 💡 Enhancements 💡
* Add cert manager 1.8.0 ([#905](https://github.com/open-telemetry/opentelemetry-operator/pull/905), [@yuriolisa](https://github.com/yuriolisa))
* updated module name and imports ([#910](https://github.com/open-telemetry/opentelemetry-operator/pull/910), [@evanli02](https://github.com/evanli02))
### 🧰 Bug fixes 🧰
* Fix docker multiarch build for operator ([#882](https://github.com/open-telemetry/opentelemetry-operator/pull/882), [@pavolloffay](https://github.com/pavolloffay))
* avoid non static labels in workload objects selector ([#849](https://github.com/open-telemetry/opentelemetry-operator/pull/849), [@DWonMtl](https://github.com/DWonMtl))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.52.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.52.0)
* [OpenTelemetry Contrib - v0.52.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.52.0)

0.51.0
-------------------
### 🚀 New components 🚀
* Choose target container injection with annotation ([#689](https://github.com/open-telemetry/opentelemetry-operator/pull/689), [@fscellos](https://github.com/fscellos))
* Fix K8s attributes values in OTEL_RESOURCE_ATTRIBUTES env var ([#864](https://github.com/open-telemetry/opentelemetry-operator/pull/864), [@mat-rumian](https://github.com/mat-rumian))
### 💡 Enhancements 💡
* Update Python auto-instrumentation versions.txt ([#867](https://github.com/open-telemetry/opentelemetry-operator/pull/867), [@mat-rumian](https://github.com/mat-rumian))
* Update Python instrumentation to 0.30b1 ([#860](https://github.com/open-telemetry/opentelemetry-operator/pull/860), [@mat-rumian](https://github.com/mat-rumian))
* Fix changelog formatting ([#863](https://github.com/open-telemetry/opentelemetry-operator/pull/863), [@pavolloffay](https://github.com/pavolloffay))
### 🧰 Bug fixes 🧰
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.51.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.51.0)
* [OpenTelemetry Contrib - v0.51.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.51.0)

0.50.0
-------------------
### 🚀 New components 🚀
* Add resource attributes to collector sidecar ([#832](https://github.com/open-telemetry/opentelemetry-operator/pull/832), [@rubenvp8510](https://github.com/rubenvp8510))
* Create serving certs for headless services on OpenShift (#818) ([#824](https://github.com/open-telemetry/opentelemetry-operator/pull/824), [@rkukura](https://github.com/rkukura))
* [targetallocator] PrometheusOperator CRD MVC ([#653](https://github.com/open-telemetry/opentelemetry-operator/pull/653), [@secustor](https://github.com/secustor))
### 💡 Enhancements 💡
* Set replicas to MaxReplicas if HPA is enabled ([#833](https://github.com/open-telemetry/opentelemetry-operator/pull/833), [@binjip978](https://github.com/binjip978))
* Update sidecar example in README ([#837](https://github.com/open-telemetry/opentelemetry-operator/pull/837), [@erichsueh3](https://github.com/erichsueh3))
### 🧰 Bug fixes 🧰
* Fix Default Image Annotations ([#842](https://github.com/open-telemetry/opentelemetry-operator/pull/842), [@goatsthatcode](https://github.com/goatsthatcode))
* Do not block pod creating on internal error in webhook ([#811](https://github.com/open-telemetry/opentelemetry-operator/pull/811), [@pavolloffay](https://github.com/pavolloffay))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.50.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.50.0)
* [OpenTelemetry Contrib - v0.50.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.50.0)

0.49.0
-------------------
### 🚀 New components 🚀
* Including new label ([#797](https://github.com/open-telemetry/opentelemetry-operator/pull/797), [@yuriolisa](https://github.com/yuriolisa))
* Add scale subresource status to the OpenTelemetryCollector CRD status ([#785](https://github.com/open-telemetry/opentelemetry-operator/pull/785), [@secat](https://github.com/secat))
### 💡 Enhancements 💡
* Set replicas to default value ([#814](https://github.com/open-telemetry/opentelemetry-operator/pull/814), [@pavolloffay](https://github.com/pavolloffay))
* Use golang 1.18 ([#786](https://github.com/open-telemetry/opentelemetry-operator/pull/786), [@pavolloffay](https://github.com/pavolloffay))
* Support nodeSelector field for non-sidecar collectors ([#789](https://github.com/open-telemetry/opentelemetry-operator/pull/789), [@jutley](https://github.com/jutley))
* Fix Missing parameter on labels function ([#809](https://github.com/open-telemetry/opentelemetry-operator/pull/809), [@yuriolisa](https://github.com/yuriolisa))
### 🧰 Bug fixes 🧰
* Check exposed svc ports ([#778](https://github.com/open-telemetry/opentelemetry-operator/pull/778), [@yuriolisa](https://github.com/yuriolisa))
* Fix panic when spec.replicas is nil ([#798](https://github.com/open-telemetry/opentelemetry-operator/pull/798), [@wei840222](https://github.com/wei840222))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.49.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.49.0)
* [OpenTelemetry Contrib - v0.49.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.49.0)

0.48.0
-------------------
* Bumped OpenTelemetry Collector to v0.48.0
* Filter out unneeded labels ([#606](https://github.com/open-telemetry/opentelemetry-operator/pull/606), [@ekarlso](https://github.com/ekarlso))
* add labels in order to make selectors unique ([#796](https://github.com/open-telemetry/opentelemetry-operator/pull/796), [@davidkarlsen](https://github.com/davidkarlsen))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.48.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.48.0)
* [OpenTelemetry Contrib - v0.48.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.48.0)

0.47.0
-------------------
* Bumped OpenTelemetry Collector to v0.47.0
* doc: customized auto-instrumentation ([#762](https://github.com/open-telemetry/opentelemetry-operator/pull/762), [@cuichenli](https://github.com/cuichenli))
* Remove v prefix from the container image version/tag ([#771](https://github.com/open-telemetry/opentelemetry-operator/pull/771), [@pavolloffay](https://github.com/pavolloffay))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.47.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.47.0)
* [OpenTelemetry Contrib - v0.47.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.47.0)

0.46.0
-------------------
* Bumped OpenTelemetry Collector to v0.46.0
* add autoscale option to enable support for Horizontal Pod Autoscaling ([#746](https://github.com/open-telemetry/opentelemetry-operator/pull/746), [@binjip978](https://github.com/binjip978))
* chore(nodejs): bump auto-instrumentations ([#763](https://github.com/open-telemetry/opentelemetry-operator/pull/763), [@mat-rumian](https://github.com/mat-rumian))
* Make operator more resiliant to etcd defrag activity ([#742](https://github.com/open-telemetry/opentelemetry-operator/pull/742), [@pavolloffay](https://github.com/pavolloffay))
#### OpenTelemetry Collector and Contrib
* [OpenTelemetry Collector - v0.46.0](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.46.0)
* [OpenTelemetry Contrib - v0.46.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.46.0)

0.45.0
-------------------
* Bumped OpenTelemetry Collector to v0.45.0
* Match pod `dnsPolicy` to `hostNetwork` config ([#691](https://github.com/open-telemetry/opentelemetry-operator/pull/691), [@gai6948](https://github.com/gai6948))
* Change container image USER to UID ([#738](https://github.com/open-telemetry/opentelemetry-operator/pull/738), [@kraman](https://github.com/kraman))
* Use OTEL collector image from GHCR ([#732](https://github.com/open-telemetry/opentelemetry-operator/pull/732), [@pavolloffay](https://github.com/pavolloffay))

0.44.0
-------------------
* Bumped OpenTelemetry Collector to v0.44.0
* Deprecate otelcol status messages ([#733](https://github.com/open-telemetry/opentelemetry-operator/pull/733), [@pavolloffay](https://github.com/pavolloffay))
* Make sure correct version of operator-sdk is always used ([#728](https://github.com/open-telemetry/opentelemetry-operator/pull/728), [@pavolloffay](https://github.com/pavolloffay))
* Storing upgrade status into events ([#707](https://github.com/open-telemetry/opentelemetry-operator/pull/707), [@yuriolisa](https://github.com/yuriolisa))
* Bump default java auto-instrumentation version to `1.11.1` ([#731](https://github.com/open-telemetry/opentelemetry-operator/pull/731), [@pavolloffay](https://github.com/pavolloffay))
* Add status fields for instrumentation kind ([#717](https://github.com/open-telemetry/opentelemetry-operator/pull/717), [@frzifus](https://github.com/frzifus))
* Add appProtocol for otlp and jaeger receiver parsers ([#704](https://github.com/open-telemetry/opentelemetry-operator/pull/704), [@binjip978](https://github.com/binjip978))
* Add SPLUNK_ env prefix support to Instrumentation kind ([#709](https://github.com/open-telemetry/opentelemetry-operator/pull/709), [@elvis-cai](https://github.com/elvis-cai))
* Fix logger in instrumentation webhook ([#698](https://github.com/open-telemetry/opentelemetry-operator/pull/698), [@pavolloffay](https://github.com/pavolloffay))

0.43.0
-------------------
* Bumped OpenTelemetry Collector to v0.43.0
* Upgrade to 0.43.0 will move the metrics CLI arguments into the config, in response to ([#680](https://github.com/open-telemetry/opentelemetry-operator/pull/680), [@yuriolisa](https://github.com/yuriolisa))
* Add unique label and selector for operator objects ([#697](https://github.com/open-telemetry/opentelemetry-operator/pull/697), [@pavolloffay](https://github.com/pavolloffay))
* Bump operator-sdk to 1.17 ([#692](https://github.com/open-telemetry/opentelemetry-operator/pull/692), [@pavolloffay](https://github.com/pavolloffay))
* Update java instrumentation to 1.10.1 ([#688](https://github.com/open-telemetry/opentelemetry-operator/pull/688), [@anuraaga](https://github.com/anuraaga))
* Update nodejs instrumentation to 0.27.0 ([#687](https://github.com/open-telemetry/opentelemetry-operator/pull/687), [@anuraaga](https://github.com/anuraaga))
* Update python instrumentation to 0.28b1 ([#686](https://github.com/open-telemetry/opentelemetry-operator/pull/686), [@anuraaga](https://github.com/anuraaga))
* Add b3, jaeger, ottrace propagators to python instrumentation ([#684](https://github.com/open-telemetry/opentelemetry-operator/pull/684), [@anuraaga](https://github.com/anuraaga))
* Add env support to instrumentation kind  ([#674](https://github.com/open-telemetry/opentelemetry-operator/pull/674), [@Duncan-tree-zhou](https://github.com/Duncan-tree-zhou))
* Fix collector config update ([#670](https://github.com/open-telemetry/opentelemetry-operator/pull/670), [@mcariapas](https://github.com/mcariapas))

0.42.0
-------------------
* Bumped OpenTelemetry Collector to v0.42.0
* Parse flags before using them in config ([#662](https://github.com/open-telemetry/opentelemetry-operator/pull/662), [@rubenvp8510](https://github.com/rubenvp8510))
* Fix port derivation ([#651](https://github.com/open-telemetry/opentelemetry-operator/pull/651), [@yuriolisa](https://github.com/yuriolisa))
* Remove publishing operator image to quay.io ([#661](https://github.com/open-telemetry/opentelemetry-operator/pull/661), [@pavolloffay](https://github.com/pavolloffay))
* Use target allocator from GHCR ([#660](https://github.com/open-telemetry/opentelemetry-operator/pull/660), [@pavolloffay](https://github.com/pavolloffay))

0.41.1
-------------------
* Add support for nodejs and python image defaulting and upgrade ([#607](https://github.com/open-telemetry/opentelemetry-operator/pull/607), [@pavolloffay](https://github.com/pavolloffay))
* Bugfix for `kubeletstats` receiver operator is exposing the service port, ignore port exposition as it is a scraper ([#558](https://github.com/open-telemetry/opentelemetry-operator/pull/558), [@mritunjaysharma394](https://github.com/mritunjaysharma394))

0.41.0
-------------------
* Bumped OpenTelemetry Collector to v0.41.0
* Support `OpenTelemetryCollector.Spec.UpgradeStrategy` with allowable values: automatic, none ([#620](https://github.com/open-telemetry/opentelemetry-operator/pull/620), [@adriankostrubiak-tomtom](https://github.com/adriankostrubiak-tomtom))
* Limit names and labels to 63 characters ([#609](https://github.com/open-telemetry/opentelemetry-operator/pull/609), [@mmatache](https://github.com/mmatache))
* Support `healthz` and `readyz` probes to controller manager ([#603](https://github.com/open-telemetry/opentelemetry-operator/pull/603), [@adriankostrubiak-tomtom](https://github.com/adriankostrubiak-tomtom))


0.40.0
-------------------
* Bumped OpenTelemetry Collector to v0.40.0
* Support K8s liveness probe to otel collector, if health_check extension is defined in otel collector config ([#574](https://github.com/open-telemetry/opentelemetry-operator/pull/574))

0.39.0
-------------------
* Bumped OpenTelemetry Collector to v0.39.0
* Upgrade path for Instrumentation kind ([#548](https://github.com/open-telemetry/opentelemetry-operator/pull/548))
* Auto-instrumentation support for python ([#532](https://github.com/open-telemetry/opentelemetry-operator/pull/532))
* Support for `PodSecurityContext` in OpenTelemetry collector ([#469](https://github.com/open-telemetry/opentelemetry-operator/pull/469))
* Java auto-instrumentation support is bumped to `1.7.2` ([#549](https://github.com/open-telemetry/opentelemetry-operator/pull/549))
* Auto-instrumentation support for nodejs ([#507](https://github.com/open-telemetry/opentelemetry-operator/pull/507))
* Sampler configuration support in instrumentation kind ([#514](https://github.com/open-telemetry/opentelemetry-operator/pull/514))

0.38.0
-------------------
* Bumped OpenTelemetry Collector to v0.38.0
* Initial support for auto-instrumentation at the moment supported only for Java ([#464](https://github.com/open-telemetry/opentelemetry-operator/pull/464), [@pavolloffay](https://github.com/pavolloffay))

0.37.1
-------------------
* Bumped OpenTelemetry Collector to v0.37.1

0.37.0
-------------------
* Bumped OpenTelemetry Collector to v0.37.0

0.36.0
-------------------
* Bumped OpenTelemetry Collector to v0.36.0
* Add `envFrom` to collector spec ([#419](https://github.com/open-telemetry/opentelemetry-operator/pull/419), [@ctison](https://github.com/ctison))
* Allow changing Pod annotations using `podAnnotations` ([#451](https://github.com/open-telemetry/opentelemetry-operator/pull/451), [@indrekj](https://github.com/indrekj))

0.35.0
-------------------
* Bumped OpenTelemetry Collector to v0.35.0
* Target Allocator implementation (Part 3 - OTEL Operator Enhancements) ([#389](https://github.com/open-telemetry/opentelemetry-operator/pull/389), [@Raul9595](https://github.com/Raul9595))
* Target Allocator implementation (Part 2 - OTEL Operator Enhancements) ([#354](https://github.com/open-telemetry/opentelemetry-operator/pull/354), [@alexperez52](https://github.com/alexperez52))

0.34.0
-------------------
* Bumped OpenTelemetry Collector to v0.34.0
* Add AWS xray receiver ([#421](https://github.com/open-telemetry/opentelemetry-operator/pull/421), [@VineethReddy02](https://github.com/VineethReddy02))
* Add syslog, tcplog and udplog receivers ([#425](https://github.com/open-telemetry/opentelemetry-operator/pull/425), [@VineethReddy02](https://github.com/VineethReddy02))
* Add splunk hec receiver ([#422](https://github.com/open-telemetry/opentelemetry-operator/pull/422), [@VineethReddy02](https://github.com/VineethReddy02))
* Add influxdb receiver ([#423](https://github.com/open-telemetry/opentelemetry-operator/pull/423), [@VineethReddy02](https://github.com/VineethReddy02))
* Added imagePullPolicy option to CRD ([#413](https://github.com/open-telemetry/opentelemetry-operator/pull/413), [@mmatache](https://github.com/mmatache))

0.33.0 (2021-08-20)
-------------------
* Bumped OpenTelemetry Collector to v0.33.0
* Add statsd receiver ([#364](https://github.com/open-telemetry/opentelemetry-operator/pull/364), [@VineethReddy02](https://github.com/VineethReddy02))
* Allow running daemonset in hostNetwork mode ([#393](https://github.com/open-telemetry/opentelemetry-operator/pull/393), [@owais](https://github.com/owais))
* Target Allocator implementation (Part 1 - OTEL Operator Enhancements) ([#351](https://github.com/open-telemetry/opentelemetry-operator/pull/351), [@]())
* Change the default port for OTLP HTTP ([#373](https://github.com/open-telemetry/opentelemetry-operator/pull/373), [@joaopgrassi](https://github.com/joaopgrassi))
* Add Kubernetes 1.22 to the test matrix ([#382](https://github.com/open-telemetry/opentelemetry-operator/pull/382), [@jpkrohling](https://github.com/jpkrohling))
* Add `protocol: TCP` value under `ports` key to avoid the known limitation for Kubernetes 1.19 ([#372](https://github.com/open-telemetry/opentelemetry-operator/pull/372), [@Saber-W](https://github.com/Saber-W))
* Add fluentforward receiver ([#367](https://github.com/open-telemetry/opentelemetry-operator/pull/367), [@VineethReddy02](https://github.com/VineethReddy02))

0.32.0
-------------------
* We skipped this release.

0.31.0 (2021-07-29)
-------------------
* Bumped OpenTelemetry Collector to v0.31.0

0.30.0 (2021-07-15)
-------------------
* Bumped OpenTelemetry Collector to v0.30.0
* Container Security Context ([#332](https://github.com/open-telemetry/opentelemetry-operator/pull/332), [@owais](https://github.com/owais))

0.29.0 (2021-06-25)
-------------------
* Bumped OpenTelemetry Collector to v0.29.0
* Add delete webhook ([#313](https://github.com/open-telemetry/opentelemetry-operator/pull/313), [@VineethReddy02](https://github.com/VineethReddy02))

0.28.0 (2021-06-12)
-------------------
* Bumped OpenTelemetry Collector to v0.28.0
* Tolerations support in OpenTelemetryCollector CRD ([#302](https://github.com/open-telemetry/opentelemetry-operator/pull/302), [@VineethReddy02](https://github.com/VineethReddy02))
* Copy desired service ports when reconciling ([#299](https://github.com/open-telemetry/opentelemetry-operator/pull/299), [@thib92](https://github.com/thib92))
* Remove the OTLP receiver legacy gRPC port(55680) references ([#293](https://github.com/open-telemetry/opentelemetry-operator/pull/293), [@mxiamxia](https://github.com/mxiamxia))

0.27.0 (2021-05-20)
-------------------
* Bumped OpenTelemetry Collector to v0.27.0

0.26.0 (2021-05-12)
-------------------
* Bumped OpenTelemetry Collector to v0.26.0

0.25.0 (2021-05-06)
-------------------
* Bumped OpenTelemetry Collector to v0.25.0

0.24.0 (2021-04-20)
-------------------
* Bumped OpenTelemetry Collector to v0.24.0 ([#251](https://github.com/open-telemetry/opentelemetry-operator/pull/251), [@jnodorp-jaconi](https://github.com/jnodorp-jaconi))
* Allow resource configuration on collector spec ([#248](https://github.com/open-telemetry/opentelemetry-operator/pull/248), [@jnodorp-jaconi](https://github.com/jnodorp-jaconi))

0.23.0 (2021-04-04)
-------------------
* Bumped OpenTelemetry Collector to v0.23.0

0.22.0 (2021-03-11)
-------------------
* Bumped OpenTelemetry Collector to v0.22.0

0.21.0 (2021-03-09)
-------------------
* Bumped OpenTelemetry Collector to v0.21.0
* Restart collector pod when config is updated ([#215](https://github.com/open-telemetry/opentelemetry-operator/pull/215), [@bhiravabhatla](https://github.com/bhiravabhatla))
* Add permissions for opentelemetry finalizer resource ([#212](https://github.com/open-telemetry/opentelemetry-operator/pull/212), [@rubenvp8510](https://github.com/rubenvp8510))
* fix: collector selection should not fail if there is a single sidecar ([#210](https://github.com/open-telemetry/opentelemetry-operator/pull/210), [@vbehar](https://github.com/vbehar))

0.20.0 (2021-02-11)
-------------------
* Bumped OpenTelemetry Collector to v0.20.0
* Add correct boundary to integer parsing ([#187](https://github.com/open-telemetry/opentelemetry-operator/pull/187), [@jpkrohling](https://github.com/jpkrohling))

0.19.0 (2021-01-27)
-------------------
* Bumped OpenTelemetry Collector to v0.19.0


0.18.1 (2021-01-25)
-------------------
* Fixed testing image from being used in non-test artifacts (fixes #170) ([#171](https://github.com/open-telemetry/opentelemetry-operator/pull/171), [@gramidt](https://github.com/gramidt))


0.18.0 (2021-01-22)
-------------------
* Bumped OpenTelemetry Collector to v0.18.0 ([#169](https://github.com/open-telemetry/opentelemetry-operator/pull/169), [@jpkrohling](https://github.com/jpkrohling))


0.17.1 (2020-12-17)
-------------------
* Set env vars correctly in workflow steps ([#152](https://github.com/open-telemetry/opentelemetry-operator/pull/152), [@jpkrohling](https://github.com/jpkrohling))
* Add permissions for leases.coordination.k8s.io ([#151](https://github.com/open-telemetry/opentelemetry-operator/pull/151), [@jpkrohling](https://github.com/jpkrohling))
* Adjust container image tags ([#148](https://github.com/open-telemetry/opentelemetry-operator/pull/148), [@jpkrohling](https://github.com/jpkrohling))

0.17.0 (2020-12-16)
-------------------
* Bumped OpenTelemetry Collector to v0.17.0 ([#144](https://github.com/open-telemetry/opentelemetry-operator/pull/144), [@jpkrohling](https://github.com/jpkrohling))
* Refactor how images are pushed ([#138](https://github.com/open-telemetry/opentelemetry-operator/pull/138), [@jpkrohling](https://github.com/jpkrohling))

0.16.0 (2020-12-03)
-------------------
* Bumped OpenTelemetry Collector to v0.16.0 ([#135](https://github.com/open-telemetry/opentelemetry-operator/pull/135), [@jpkrohling](https://github.com/jpkrohling))
* Fix image prefix for release image ([#133](https://github.com/open-telemetry/opentelemetry-operator/pull/133), [@jpkrohling](https://github.com/jpkrohling))
* Explicitly set Service Port Protocol for Jaeger Receivers ([#117](https://github.com/open-telemetry/opentelemetry-operator/pull/117), [@KingJ](https://github.com/KingJ))

_Note: The default port for the OTLP receiver has been changed from 55680 to 4317. To keep compatibility with your existing workload, the operator is now generating a service with the two port numbers by default. Both have 4317 as the target port._

0.15.0 (2020-11-27)
-------------------
* Bumped OpenTelemetry Collector to v0.15.0 ([#131](https://github.com/open-telemetry/opentelemetry-operator/pull/131), [@jpkrohling](https://github.com/jpkrohling)) 

0.14.0 (2020-11-09)
-------------------
* Bumped OpenTelemetry Collector to v0.14.0 ([#112](https://github.com/open-telemetry/opentelemetry-operator/pull/112), [@jpkrohling](https://github.com/jpkrohling)) 

_Note: The `tailsampling` processor was moved to the contrib repository, requiring a manual intervention in case this processor is being used: either replace the image with the contrib one (v0.14.0, which includes this processor), or remove the processor._

0.13.0 (2020-10-22)
-------------------

* Bumped OpenTelemetry Collector to v0.13.0 ([#101](https://github.com/open-telemetry/opentelemetry-operator/pull/101), [@dengliming](https://github.com/dengliming)) 
* Allow for spec.Env to be set on the OTEL Collector Spec ([#94](https://github.com/open-telemetry/opentelemetry-operator/pull/94), [@ekarlso](https://github.com/ekarlso))

_Note: The `groupbytrace` processor was moved to the contrib repository, requiring a manual intervention in case this processor is being used: either replace the image with the contrib one (v0.13.1, which includes this processor), or remove the processor._

0.12.0 (2020-10-12)
-------------------

* Bumped OpenTelemetry Collector to v0.12.0 ([#81](https://github.com/open-telemetry/opentelemetry-operator/pull/81), [@jpkrohling](https://github.com/jpkrohling))
* Remove use of deprecated controller runtime log API ([#78](https://github.com/open-telemetry/opentelemetry-operator/pull/78), [@bvwells](https://github.com/bvwells))

0.11.0 (2020-09-30)
-------------------

- Initial release after the migration to `kubebuilder`
- Support for OpenTelemetry Collector v0.11.0
- Features:
  - Provisioning of an OpenTelemetry Collector based on the CR definition
  - Sidecar injected via webhook
  - Deployment modes: `daemonset`, `deployment`, `sidecar`
  - Automatic upgrade between collector versions
- CRs from the older version should still work with this operator
