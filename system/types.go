package system

import (
	"io/fs"

	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/config/readers/file"
	"github.com/cantara/nerthus2/executors/ansible"
	"github.com/cantara/nerthus2/system/service"
)

type Environment struct {
	Name          string                  `yaml:"name"`
	Domain        string                  `yaml:"domain"`
	Vars          Vars                    `yaml:"vars"`
	Systems       []string                `yaml:"systems"`
	Nerthus       string                  `yaml:"nerthus_host"`
	Visuale       string                  `yaml:"visuale_host"`
	OSName        string                  `yaml:"os_name"`
	OSArch        string                  `yaml:"os_arch"`
	InstanceType  string                  `yaml:"instance_type"`
	Roles         map[string]ansible.Role `yaml:",omitempty"`
	FS            fs.FS                   `yaml:",omitempty"`
	Dir           string                  `yaml:",omitempty"`
	SystemConfigs map[string]System       `yaml:",omitempty"`
}

type RoutingMethod string

const (
	RoutingPath RoutingMethod = "path"
	RoutingHost RoutingMethod = "host"
)

type System struct {
	Name              string                  `yaml:"name"`
	Domain            string                  `yaml:"domain"`
	Vars              Vars                    `yaml:"vars"`
	Clusters          []*Cluster              `yaml:"clusters"`
	Scope             string                  `yaml:"scope"`
	VPC               string                  `yaml:"vpc"`
	Key               string                  `yaml:"key"`
	RoutingMethod     RoutingMethod           `yaml:"routing_method"`
	Loadbalancer      string                  `yaml:"loadbalancer"`
	LoadbalancerGroup string                  `yaml:"loadbalancerGroup"`
	OSName            string                  `yaml:"os_name"`
	OSArch            string                  `yaml:"os_arch"`
	InstanceType      string                  `yaml:"instance_type"`
	CIDR              string                  `yaml:"cidr_base"`
	Zone              string                  `yaml:"zone"`
	Roles             map[string]ansible.Role `yaml:",omitempty"`
	FS                fs.FS                   `yaml:",omitempty"`
	Dir               string                  `yaml:",omitempty"`
}

type Vars map[string]any

type Cluster struct {
	Name          string            `yaml:"name"`
	IAM           string            `yaml:"iam"`
	OSName        string            `yaml:"os_name"`
	OSArch        string            `yaml:"os_arch"`
	Arch          ami.Arch          `yaml:",omitempty"`
	InstanceType  string            `yaml:"instance_type"`
	Services      []*Service        `yaml:"services"`
	Vars          Vars              `yaml:"vars"`
	Expose        map[string]int    `yaml:"expose,omitempty"`
	Playbook      string            `yaml:"playbook,omitempty"`
	Override      map[string]string `yaml:"override,omitempty"`
	Internal      bool              `yaml:"internal"` //This could probably be handled with looking for webserverport on all services in the cluster
	NumberOfNodes int               `yaml:"number_of_nodes"`
	NodeNames     []string          `yaml:"node_names"`
	DNSRoot       string            `yaml:"dns_root"`
	SecurityGroup string            `yaml:"security_group"`
	TargetGroup   string            `yaml:"target_group"`
	//Dirs               *map[string]string          `yaml:"dirs,omitempty"` //Keeping these as it might be helpfull to copy files to the ec2 user aswell
	//Files              *map[string]file.File       `yaml:"files,omitempty"` //Keeping these as it might be helpfull to copy files to the ec2 user aswell
	Roles              map[string]ansible.Role     `yaml:",omitempty"`
	SecurityGroupRules []ansible.SecurityGroupRule `yaml:",omitempty"`
	ClusterInfo        map[string]ClusterInfo      `yaml:",omitempty"`
	Generated          bool                        `yaml:",omitempty"`
}

func (c Cluster) HasWebserverPort() bool {
	for _, serv := range c.Services {
		if serv.WebserverPort != nil {
			return true
		}
	}
	return false
}

func (c Cluster) GetWebserverPort() int {
	for _, serv := range c.Services {
		if serv.WebserverPort != nil {
			return *serv.WebserverPort
		}
	}
	return -1
}

func (c Cluster) HasFrontend() bool {
	for _, serv := range c.Services {
		if serv.ServiceInfo.Requirements.IsFrontend {
			return true
		}
	}
	return false
}

func (c Cluster) IsClusterAble() bool {
	for _, serv := range c.Services {
		if serv.ServiceInfo.Requirements.NotClusterAble {
			return false
		}
	}
	return true
}

type Service struct {
	Name          string                `yaml:"name"`
	Vars          Vars                  `yaml:"vars"`
	Local         string                `yaml:"local,omitempty"`
	Git           string                `yaml:"git,omitempty"`
	Branch        string                `yaml:"branch,omitempty"`
	WebserverPort *int                  `yaml:"webserver_port,omitempty"`
	Properties    *string               `yaml:"properties,omitempty"`
	Dirs          *map[string]string    `yaml:"dirs,omitempty"`
	Files         *map[string]file.File `yaml:"files,omitempty"`
	ServiceInfo   *service.Service      `yaml:",omitempty"`
}

type ClusterInfo struct {
	Name  string         `yaml:"name"`
	Ports map[string]int `yaml:"ports"`
}
