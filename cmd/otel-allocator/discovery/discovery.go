package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kit/log"
	"github.com/otel-allocator/allocation"
	"github.com/otel-allocator/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/azure"
	"github.com/prometheus/prometheus/discovery/consul"
	"github.com/prometheus/prometheus/discovery/digitalocean"
	"github.com/prometheus/prometheus/discovery/dns"
	"github.com/prometheus/prometheus/discovery/eureka"
	"github.com/prometheus/prometheus/discovery/file"
	"github.com/prometheus/prometheus/discovery/gce"
	"github.com/prometheus/prometheus/discovery/hetzner"
	"github.com/prometheus/prometheus/discovery/http"
	"github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/discovery/linode"
	"github.com/prometheus/prometheus/discovery/marathon"
	"github.com/prometheus/prometheus/discovery/openstack"
	"github.com/prometheus/prometheus/discovery/scaleway"
	"github.com/prometheus/prometheus/discovery/triton"
	yaml "gopkg.in/yaml.v2"
)

type Manager struct {
	manager *discovery.Manager
	logger  log.Logger

	close chan struct{}
}

func NewManager(ctx context.Context, logger log.Logger, options ...func(*discovery.Manager)) *Manager {
	manager := discovery.NewManager(ctx, logger, options...)
	go func() {
		if err := manager.Run(); err != nil {
			logger.Log("Discovery manager failed", err)
		}
	}()
	return &Manager{
		manager: manager,
		logger:  logger,
		close:   make(chan struct{}),
	}
}

func (m *Manager) ApplyConfig(cfg config.Config) error {
	discoveryCfg := make(map[string]discovery.Configs)

	for _, scrapeConfig := range cfg.Config.ScrapeConfigs {
		discoveryConfigs := discovery.Configs{}
		for name, sd := range scrapeConfig {
			if strings.HasSuffix(name, "_sd_configs") {
				sdYAML, _ := yaml.Marshal(sd)
				switch name {
				case "azure_sd_configs":
					sdConfig := []azure.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling azure sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "consul_sd_configs":
					sdConfig := []consul.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling consul sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "digitalocean_sd_configs":
					sdConfig := []digitalocean.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling digitalocean sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "docker_sd_configs":
				// 	sdConfig := []docker.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling docker sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				// case "dockerswarm_sd_configs":
				// 	sdConfig := []dockerswarm.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling dockerswarm sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "dns_sd_configs":
					sdConfig := []dns.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling dns sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "ec2_sd_configs":
				// 	sdConfig := []ec2.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling ec2 sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "openstack_sd_configs":
					sdConfig := []openstack.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling openstack sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "file_sd_configs":
					sdConfig := []file.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling file sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "gce_sd_configs":
					sdConfig := []gce.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling gce sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "hetzner_sd_configs":
					sdConfig := []hetzner.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling hetzner sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "http_sd_configs":
					sdConfig := []http.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling http sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "kubernetes_sd_configs":
					sdConfig := []kubernetes.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling kubernetes sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "lightsail_sd_configs":
				// 	sdConfig := []lightsail.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling lightsail sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "linode_sd_configs":
					sdConfig := []linode.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling linode sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "marathon_sd_configs":
					sdConfig := []marathon.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling marathon sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "nerve_sd_configs":
				// 	sdConfig := []nerve.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error nerve azure sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				// case "serverset_sd_configs":
				// 	sdConfig := []serverset.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error serverset azure sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "triton_sd_configs":
					sdConfig := []triton.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling triton sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "eureka_sd_configs":
					sdConfig := []eureka.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling eureka sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "scaleway_sd_configs":
					sdConfig := []scaleway.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling scaleway sd config: %s", err)
					}
					for index := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				}
			} else if name == "static_configs" {
				staticYAML, _ := yaml.Marshal(sd)
				staticConfig := discovery.StaticConfig{}
				err := yaml.UnmarshalStrict(staticYAML, &staticConfig)
				if err != nil {
					fmt.Printf("error unmarshalling static config: %s", err)
				}
				discoveryConfigs = append(discoveryConfigs, staticConfig)
			}
		}
		discoveryCfg[scrapeConfig["job_name"].(string)] = discoveryConfigs
	}
	return m.manager.ApplyConfig(discoveryCfg)
}

func (m *Manager) Watch(fn func(targets []allocation.TargetItem)) {
	go func() {
		for {
			select {
			case <-m.close:
				return
			case tsets := <-m.manager.SyncCh():
				targets := []allocation.TargetItem{}

				for jobName, tgs := range tsets {
					for _, tg := range tgs {
						for _, t := range tg.Targets {
							targets = append(targets, allocation.TargetItem{
								JobName:   jobName,
								TargetURL: string(t[model.AddressLabel]),
								Label:     tg.Labels,
							})
						}
					}
				}
				fn(targets)
			}
		}
	}()
}

func (m *Manager) Close() {
	close(m.close)
}
