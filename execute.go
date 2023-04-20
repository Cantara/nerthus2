package main

import (
	"context"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	"github.com/cantara/nerthus2/executors"
	"github.com/cantara/nerthus2/executors/ansible/generators"
	"github.com/cantara/nerthus2/message"
	"os"
	"strings"
)

/*
	var bootstrapVars *properties.BootstrapVars
	if bootstrap {
		bootstrapVars = &properties.BootstrapVars{
			GitToken: gitToken,
			GitRepo:  gitRepo,
			EnvName:  bootstrapEnv,
		}
	}
*/

var baseFS = os.DirFS(".")

func ExecuteEnv(env string) {
	envConf, err := config.ReadFullEnv(env, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if bootstrap && strings.ToLower(systemConf.Name) != "nerthus" {
			log.Info("skipping systemConf while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name)
			continue
		}
		for _, cluster := range systemConf.Clusters {
			if bootstrap && strings.ToLower(cluster.Name) != "nerthus" { //FIXME: This logic is flawed
				log.Info("skipping cluster while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name, "cluster", cluster.Name)
				continue
			}
			log.Info("executing cluster", "env", envConf.Name, "system", systemConf.Name, "cluster", cluster.Name, "overrides", cluster.Override)

			for _, service := range cluster.Services {
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *cluster, *service)
				for nodeNum, nodeName := range cluster.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*cluster, nodeNum, serviceVars) //, bootstrapVars)
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

				clusterVars := config.ClusterProvisioningVars(envConf, systemConf, *cluster, bootstrap)
				retChan := executors.ExecuteClusterProvisioning(envConf.Dir, clusterVars, context.Background())
				for status := range retChan {
					log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
				}
			}
		}

		systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
		retChan := executors.ExecuteLoadbalancerProvisioning(envConf.Dir, systemLoadbalancerVars, context.Background())
		for status := range retChan {
			log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
		}
	}
}

func ExecuteSys(env, sys string) {
	if bootstrap {
		log.Fatal("can't bootstrap a single systemConf", "env", env, "system", sys)
		//Might want to allow this
	}
	envConf, err := config.ReadFullEnv(env, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if strings.ToLower(systemConf.Name) != sys {
			continue
		}
		for _, cluster := range systemConf.Clusters {
			if bootstrap && strings.ToLower(cluster.Name) != "nerthus" { //FIXME: This logic is flawed
				log.Info("skipping cluster while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name, "cluster", cluster.Name)
				continue
			}
			log.Info("executing cluster", "env", envConf.Name, "system", systemConf.Name, "cluster", cluster.Name, "overrides", cluster.Override)

			for _, service := range cluster.Services {
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *cluster, *service)
				for nodeNum, nodeName := range cluster.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*cluster, nodeNum, serviceVars) //, bootstrapVars)
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

				clusterVars := config.ClusterProvisioningVars(envConf, systemConf, *cluster, bootstrap)
				retChan := executors.ExecuteClusterProvisioning(envConf.Dir, clusterVars, context.Background())
				for status := range retChan {
					log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
				}
			}
		}
		systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
		retChan := executors.ExecuteLoadbalancerProvisioning(envConf.Dir, systemLoadbalancerVars, context.Background())
		for status := range retChan {
			log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
		}
	}
}

func ExecuteClust(env, sys, cluster string) {
	if bootstrap {
		log.Fatal("can't bootstrap a single service", "env", env, "system", sys, "cluster", cluster)
		//Might want to allow this
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
				serviceVars := config.ServiceProvisioningVars(envConf, systemConf, *clusterConf, *service)
				for nodeNum, nodeName := range clusterConf.NodeNames {
					serviceNodeVars := config.ServiceNodeVars(*clusterConf, nodeNum, serviceVars) //, bootstrapVars)
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

				clusterVars := config.ClusterProvisioningVars(envConf, systemConf, *clusterConf, bootstrap)
				retChan := executors.ExecuteClusterProvisioning(envConf.Dir, clusterVars, context.Background())
				for status := range retChan {
					log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
				}
			}
		}
		systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
		retChan := executors.ExecuteLoadbalancerProvisioning(envConf.Dir, systemLoadbalancerVars, context.Background())
		for status := range retChan {
			log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
		}
	}
}

func ExecuteServ(env, sys, cluster, serv string) {
	if bootstrap {
		log.Fatal("can't bootstrap a single service", "env", env, "system", sys, "cluster", cluster, "service", serv)
		//Might want to allow this
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
					serviceNodeVars := config.ServiceNodeVars(*clusterConf, nodeNum, serviceVars) //, bootstrapVars)
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
