package main

import (
	"context"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	"github.com/cantara/nerthus2/config/systems"
	"github.com/cantara/nerthus2/executors"
	"github.com/cantara/nerthus2/executors/ansible/executor"
	"github.com/cantara/nerthus2/executors/ansible/generators"
	"github.com/cantara/nerthus2/message"
	"github.com/cantara/nerthus2/system"
	"path/filepath"
	"strings"
)

func ReadFullEnvConfig(env string) (envConf system.Environment, err error) {
	builtinRoles, err := systems.BuiltinRoles(EFS)
	if err != nil {
		log.WithError(err).Fatal("while getting builtin roles")
	}
	envConf, err = systems.Environment(env, builtinRoles)
	if err != nil {
		log.WithError(err).Fatal("while getting environment config")
	}
	for _, systemName := range envConf.Systems {
		systemConf, err := systems.System(envConf, systemName)
		if err != nil {
			log.WithError(err).Error("while getting system config")
			continue
		}
		envConf.SystemConfigs[systemName] = systemConf
		for _, service := range systemConf.Services {
			err = systems.Service(systemConf, service)
			if err != nil {
				log.WithError(err).Error("while getting service config")
				continue
			}
			if bootstrap && strings.ToLower(service.ServiceInfo.Name) != "nerthus" {
				//log.Info("skipping service while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name, "service", service.Name)
				continue
			}
			//log.Info("executing service", "env", envConf.Name, "system", systemConf.Name, "service", service.Name, "overrides", service.Override)

		}
	}
	return
}

func ExecuteEnv(env string) {
	envConf, err := ReadFullEnvConfig(env)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	var bootstrapVars *config.BootstrapVars
	if bootstrap {
		bootstrapVars = &config.BootstrapVars{
			GitToken: gitToken,
			GitRepo:  gitRepo,
			EnvName:  bootstrapEnv,
		}
	}
	for _, systemConf := range envConf.SystemConfigs {
		if bootstrap && strings.ToLower(systemConf.Name) != "nerthus" {
			log.Info("skipping systemConf while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name)
			continue
		}
		for _, service := range systemConf.Services {
			if bootstrap && strings.ToLower(service.ServiceInfo.Name) != "nerthus" {
				log.Info("skipping service while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name, "service", service.Name)
				continue
			}
			log.Info("executing service", "env", envConf.Name, "system", systemConf.Name, "service", service.Name, "overrides", service.Override)

			serviceProvisioningVars := config.ServiceProvisioningVars(envConf, systemConf, *service, bootstrap)
			for nodeNum, nodeName := range service.NodeNames {
				nodeBootstrapVars := config.NodeBootstrapVars(envConf, systemConf, *service, nodeNum, serviceProvisioningVars, bootstrapVars)
				nodeBootstrapPlayYaml, err := generators.PlayToYaml(generators.GenerateNodePlay(*service, nodeBootstrapVars))
				if err != nil {
					log.WithError(err).Error("while trying to create playbook yaml")
					continue
				}

				if bootstrap {
					err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, nodeBootstrapPlayYaml, bootstrap)
					if err != nil {
						log.WithError(err).Error("while trying to write playbook yaml")
						continue
					}
				} else {
					ha, ok := hostActions.Get(nodeName)
					if !ok {
						ha = make(chan message.Action, 10)
						hostActions.Set(nodeName, ha)
					}
					ha <- message.Action{
						Action:          message.Playbook,
						AnsiblePlaybook: nodeBootstrapPlayYaml,
					}
				}

				nodeProvisioningVars := config.NodeProvisioningVars(*service, nodeNum, serviceProvisioningVars)
				serviceProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateServiceProvisioningPlay(*service, nodeProvisioningVars))
				if err != nil {
					log.WithError(err).Error("while trying to create playbook yaml")
					continue
				}
				err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, serviceProvisioningPlayYaml, false)
				if err != nil {
					log.WithError(err).Error("while trying to write playbook yaml")
					continue
				}
			}

			retChan := executors.ExecuteService(envConf.Dir, serviceProvisioningVars, context.Background())
			for status := range retChan {
				log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
			}
		}

		systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
		retChan := executors.ExecuteLoadbalancer(envConf.Dir, systemLoadbalancerVars, context.Background())
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
	envConf, err := ReadFullEnvConfig(env)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if strings.ToLower(systemConf.Name) != sys {
			continue
		}
		for _, service := range systemConf.Services {
			log.Info("executing service", "env", envConf.Name, "system", systemConf.Name, "service", service.Name, "overrides", service.Override)

			serviceProvisioningVars := config.ServiceProvisioningVars(envConf, systemConf, *service, bootstrap)
			for nodeNum, nodeName := range service.NodeNames {
				nodeBootstrapVars := config.NodeBootstrapVars(envConf, systemConf, *service, nodeNum, serviceProvisioningVars, nil)
				nodeBootstrapPlayYaml, err := generators.PlayToYaml(generators.GenerateNodePlay(*service, nodeBootstrapVars))
				if err != nil {
					log.WithError(err).Error("while trying to create playbook yaml")
					continue
				}

				if bootstrap {
					err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, nodeBootstrapPlayYaml, bootstrap)
					if err != nil {
						log.WithError(err).Error("while trying to write playbook yaml")
						continue
					}
				} else {
					ha, ok := hostActions.Get(nodeName)
					if !ok {
						ha = make(chan message.Action, 10)
						hostActions.Set(nodeName, ha)
					}
					ha <- message.Action{
						Action:          message.Playbook,
						AnsiblePlaybook: nodeBootstrapPlayYaml,
					}
				}

				nodeProvisioningVars := config.NodeProvisioningVars(*service, nodeNum, serviceProvisioningVars)
				serviceProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateServiceProvisioningPlay(*service, nodeProvisioningVars))
				if err != nil {
					log.WithError(err).Error("while trying to create playbook yaml")
					continue
				}
				err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, serviceProvisioningPlayYaml, false)
				if err != nil {
					log.WithError(err).Error("while trying to write playbook yaml")
					continue
				}
			}

			retChan := executors.ExecuteService(envConf.Dir, serviceProvisioningVars, context.Background())
			for status := range retChan {
				log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
			}
		}

		systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
		retChan := executors.ExecuteLoadbalancer(envConf.Dir, systemLoadbalancerVars, context.Background())
		for status := range retChan {
			log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
		}
	}
}

func ExecuteServ(env, sys, serv string) {
	if bootstrap {
		log.Fatal("can't bootstrap a single service", "env", env, "system", sys, "service", serv)
		//Might want to allow this
	}
	envConf, err := ReadFullEnvConfig(env)
	if err != nil {
		log.WithError(err).Fatal("while reading env config")
	}
	for _, systemConf := range envConf.SystemConfigs {
		if strings.ToLower(systemConf.Name) != sys {
			continue
		}
		for _, service := range systemConf.Services {
			if strings.ToLower(service.Name) != serv {
				continue
			}
			log.Info("executing service", "env", envConf.Name, "system", systemConf.Name, "service", service.Name, "overrides", service.Override)

			serviceProvisioningVars := config.ServiceProvisioningVars(envConf, systemConf, *service, bootstrap)
			for nodeNum, nodeName := range service.NodeNames {
				nodeBootstrapVars := config.NodeBootstrapVars(envConf, systemConf, *service, nodeNum, serviceProvisioningVars, nil)
				nodeBootstrapPlayYaml, err := generators.PlayToYaml(generators.GenerateNodePlay(*service, nodeBootstrapVars))
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
					Action:          message.Playbook,
					AnsiblePlaybook: nodeBootstrapPlayYaml,
				}
			}
		}
	}
}
