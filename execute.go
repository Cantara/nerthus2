package main

import (
	"os"
	"strings"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	"github.com/cantara/nerthus2/executors/ansible/generators"
	"github.com/cantara/nerthus2/message"
	"github.com/cantara/nerthus2/system"
)

var baseFS = os.DirFS(".")

type Provision func(env system.Environment)
type ProvisionSystem func(sys system.System, env, nerthus, visuale string)
type ProvisionCluster func(clust system.Cluster, sys system.System, env, nerthus, visuale string)

func ExecuteEnv(env string, prov Provision) (id string) {
	envConf, err := config.ReadFullEnv(env, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	if bootstrap {
		envConf.SystemConfigs = map[string]system.System{"nerthus": envConf.SystemConfigs["nerthus"]}
	}
	prov(envConf)
	for _, systemConf := range envConf.SystemConfigs {
		for _, cluster := range systemConf.Clusters {
			log.Info("executing cluster", "env", envConf.Name, "system", systemConf.Name, "cluster", cluster.Name, "overrides", cluster.Override)
			for _, service := range cluster.Services {
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *cluster, *service)
				for nodeNum, nodeName := range cluster.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*cluster, nodeNum, serviceVars)
					serviceProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateServicePlay(*cluster, *service, serviceNodeVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					ha, ok := hostActions.Get(nodeName)
					if !ok {
						ha = make(chan message.Action, 10)
						hostActions.Set(nodeName, ha)
					}
					ha <- message.Action{
						Action: message.Playbook,
						Data:   serviceProvisioningPlayYaml,
					}
				}
			}
		}
	}
	return
}

func ExecuteSys(env, sys string, prov ProvisionSystem) {
	if bootstrap {
		log.Fatal("can't bootstrap a single systemConf", "env", env, "system", sys)
	}
	envConf, err := config.ReadFullEnv(env, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if strings.ToLower(systemConf.Name) != sys {
			continue
		}
		prov(systemConf, envConf.Name, envConf.Nerthus, envConf.Visuale)
		for _, cluster := range systemConf.Clusters {
			log.Info("executing cluster", "env", envConf.Name, "system", systemConf.Name, "cluster", cluster.Name, "overrides", cluster.Override)

			for _, service := range cluster.Services {
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *cluster, *service)
				for nodeNum, nodeName := range cluster.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*cluster, nodeNum, serviceVars)
					serviceProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateServicePlay(*cluster, *service, serviceNodeVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					ha, ok := hostActions.Get(nodeName)
					if !ok {
						ha = make(chan message.Action, 10)
						hostActions.Set(nodeName, ha)
					}
					ha <- message.Action{
						Action: message.Playbook,
						Data:   serviceProvisioningPlayYaml,
					}
				}
			}
		}
	}
}

func ExecuteCluster(env, sys, cluster string, prov ProvisionCluster) (id string) {
	if bootstrap {
		log.Fatal("can't bootstrap a single service", "env", env, "system", sys, "cluster", cluster)
	}
	envConf, err := config.ReadFullEnv(env, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if strings.ToLower(systemConf.Name) != sys {
			continue
		}
		for _, clusterConf := range systemConf.Clusters {
			if strings.ToLower(clusterConf.Name) != cluster {
				continue
			}
			log.Info("executing cluster", "env", envConf.Name, "system", systemConf.Name, "cluster", clusterConf.Name, "overrides", clusterConf.Override)
			prov(*clusterConf, systemConf, envConf.Name, envConf.Nerthus, envConf.Visuale)
			for _, service := range clusterConf.Services {
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *clusterConf, *service)
				for nodeNum, nodeName := range clusterConf.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*clusterConf, nodeNum, serviceVars)
					serviceProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateServicePlay(*clusterConf, *service, serviceNodeVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					ha, ok := hostActions.Get(nodeName)
					if !ok {
						ha = make(chan message.Action, 10)
						hostActions.Set(nodeName, ha)
					}
					ha <- message.Action{
						Action: message.Playbook,
						Data:   serviceProvisioningPlayYaml,
					}
				}
			}
		}
	}
	return
}

func ExecuteServ(env, sys, cluster, serv string) {
	if bootstrap {
		log.Fatal("can't bootstrap a single service", "env", env, "system", sys, "cluster", cluster, "service", serv)
	}
	envConf, err := config.ReadFullEnv(env, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if strings.ToLower(systemConf.Name) != sys {
			continue
		}
		for _, clusterConf := range systemConf.Clusters {
			if strings.ToLower(clusterConf.Name) != cluster {
				continue
			}
			log.Info("executing cluster", "env", envConf.Name, "system", systemConf.Name, "cluster", clusterConf.Name, "overrides", clusterConf.Override)

			for _, service := range clusterConf.Services {
				if strings.ToLower(service.Name) != serv {
					continue
				}
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *clusterConf, *service)
				for nodeNum, nodeName := range clusterConf.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*clusterConf, nodeNum, serviceVars)
					serviceProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateServicePlay(*clusterConf, *service, serviceNodeVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					ha, ok := hostActions.Get(nodeName)
					if !ok {
						ha = make(chan message.Action, 10)
						hostActions.Set(nodeName, ha)
					}
					ha <- message.Action{
						Action: message.Playbook,
						Data:   serviceProvisioningPlayYaml,
					}
				}
			}
		}
	}
}
