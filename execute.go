package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers"
	"github.com/cantara/nerthus2/config"
	"github.com/cantara/nerthus2/executors/ansible/generators"
	"github.com/cantara/nerthus2/message"
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

func ExecuteEnv(env string, e workers.Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client, resultChan chan<- string) {
	defer close(resultChan)
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

				//numNodes, port int, arch ami.Arch, imageName, serviceType, path, network, cluster, system, env, size, nerthus, visuale, domain string, e Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client)
				workers.Deployment(cluster.NodeNames, *service.WebserverPort, cluster.Arch, cluster.OSName, service.ServiceInfo.ServiceType, fmt.Sprintf("/%s", strings.ToLower(service.ServiceInfo.Name)), systemConf.CIDR, cluster.ClusterName, systemConf.Name, envConf.Name, cluster.InstanceType, envConf.Nerthus, envConf.Visuale, systemConf.Domain, e, e2, elb, rc, cc)
			}

			/*
				clusterVars := config.ClusterProvisioningVars(envConf, systemConf, *cluster, bootstrap)
				for nodeNum, nodeName := range cluster.NodeNames {
					nodeProvisioningVars := config.NodeProvisioningVars(*cluster, nodeNum, clusterVars)
					nodeProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateNodeProvisioningPlay(*cluster, nodeProvisioningVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, nodeProvisioningPlayYaml, false)
					if err != nil {
						log.WithError(err).Error("while trying to write playbook yaml")
						continue
					}
				}
				retChan := executors.ExecuteClusterProvisioning(envConf.Dir, clusterVars, context.Background())
				for status := range retChan {
					if resultChan != nil {
						resultChan <- status
					}
					log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
				}
			*/
		}
		/*
			systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
			retChan := executors.ExecuteLoadbalancerProvisioning(envConf.Dir, systemLoadbalancerVars, context.Background())
			for status := range retChan {
				if resultChan != nil {
					resultChan <- status
				}
				log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
			}
		*/
	}
}

func ExecuteSys(env, sys string, e workers.Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client, resultChan chan<- string) {
	defer close(resultChan)
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
				workers.Deployment(cluster.NodeNames, *service.WebserverPort, cluster.Arch, cluster.OSName, service.ServiceInfo.ServiceType, fmt.Sprintf("/%s", strings.ToLower(service.ServiceInfo.Name)), systemConf.CIDR, cluster.ClusterName, systemConf.Name, envConf.Name, cluster.InstanceType, envConf.Nerthus, envConf.Visuale, systemConf.Domain, e, e2, elb, rc, cc)
			}

			/*
				clusterVars := config.ClusterProvisioningVars(envConf, systemConf, *cluster, bootstrap)
				for nodeNum, nodeName := range cluster.NodeNames {
					nodeProvisioningVars := config.NodeProvisioningVars(*cluster, nodeNum, clusterVars)
					nodeProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateNodeProvisioningPlay(*cluster, nodeProvisioningVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, nodeProvisioningPlayYaml, false)
					if err != nil {
						log.WithError(err).Error("while trying to write playbook yaml")
						continue
					}
				}
				retChan := executors.ExecuteClusterProvisioning(envConf.Dir, clusterVars, context.Background())
				for status := range retChan {
					resultChan <- status
					log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
				}
			*/
		}
		/*
			systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
			retChan := executors.ExecuteLoadbalancerProvisioning(envConf.Dir, systemLoadbalancerVars, context.Background())
			for status := range retChan {
				resultChan <- status
				log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
			}
		*/
	}
}

func ExecuteCluster(env, sys, cluster string, e workers.Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client, resultChan chan<- string) {
	defer close(resultChan)
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
					workers.Deployment(clusterConf.NodeNames, *service.WebserverPort, clusterConf.Arch, clusterConf.OSName, service.ServiceInfo.ServiceType, fmt.Sprintf("/%s", strings.ToLower(service.ServiceInfo.Name)), systemConf.CIDR, clusterConf.ClusterName, systemConf.Name, envConf.Name, clusterConf.InstanceType, envConf.Nerthus, envConf.Visuale, systemConf.Domain, e, e2, elb, rc, cc)
				}
			}

			/*
				clusterVars := config.ClusterProvisioningVars(envConf, systemConf, *clusterConf, bootstrap)
				for nodeNum, nodeName := range clusterConf.NodeNames {
					nodeProvisioningVars := config.NodeProvisioningVars(*clusterConf, nodeNum, clusterVars)
					nodeProvisioningPlayYaml, err := generators.PlayToYaml(generators.GenerateNodeProvisioningPlay(*clusterConf, nodeProvisioningVars))
					if err != nil {
						log.WithError(err).Error("while trying to create playbook yaml")
						continue
					}
					err = executor.WriteNodePlay(filepath.Clean(envConf.Dir+"/ansible/nodes"), nodeName, nodeProvisioningPlayYaml, false)
					if err != nil {
						log.WithError(err).Error("while trying to write playbook yaml")
						continue
					}
				}
				retChan := executors.ExecuteClusterProvisioning(envConf.Dir, clusterVars, context.Background())
				for status := range retChan {
					resultChan <- status
					log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
				}
			*/
		}
		/*
			systemLoadbalancerVars := config.SystemLoadbalancerVars(envConf, systemConf)
			retChan := executors.ExecuteLoadbalancerProvisioning(envConf.Dir, systemLoadbalancerVars, context.Background())
			for status := range retChan {
				resultChan <- status
				log.WithError(status.Err).Info("executed", "task", status.Name, "status", status.Status, "msg", status.Message, "cmd", status.Command)
			}
		*/
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
