package system

import (
	"github.com/cantara/nerthus2/ansible"
	"github.com/cantara/nerthus2/system/service"
	"io/fs"
)

type Environment struct {
	Name          string                  `yaml:"name"`
	Cert          string                  `yaml:"certificate_arn"`
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

type System struct {
	Name              string                  `yaml:"name"`
	Vars              Vars                    `yaml:"vars"`
	Services          []*Service              `yaml:"services"`
	Scope             string                  `yaml:"scope"`
	VPC               string                  `yaml:"vpc"`
	Key               string                  `yaml:"key"`
	Loadbalancer      string                  `yaml:"loadbalancer"`
	LoadbalancerGroup string                  `yaml:"loadbalancerGroup"`
	OSName            string                  `yaml:"os_name"`
	OSArch            string                  `yaml:"os_arch"`
	InstanceType      string                  `yaml:"instance_type"`
	CIDR              string                  `yaml:"cidr_base"`
	Zone              string                  `yaml:"zone"`
	Roles             map[string]ansible.Role `yaml:",omitempty"`
	Dir               string                  `yaml:",omitempty"`
}

type Vars map[string]any

type Service struct {
	Name               string                      `yaml:"name"`
	Vars               Vars                        `yaml:"vars"`
	Expose             []int                       `yaml:"expose,omitempty"`
	Playbook           string                      `yaml:"playbook,omitempty"`
	Local              string                      `yaml:"local,omitempty"`
	Git                string                      `yaml:"git,omitempty"`
	Branch             string                      `yaml:"branch,omitempty"`
	Override           map[string]string           `yaml:"override,omitempty"`
	Internal           bool                        `yaml:"internal"`
	NumberOfNodes      int                         `yaml:"number_of_nodes"`
	NodeNames          []string                    `yaml:"node_names"`
	ClusterName        string                      `yaml:"cluster_name"`
	SecurityGroup      string                      `yaml:"security_group"`
	TargetGroup        string                      `yaml:"target_group"`
	IAM                string                      `yaml:"iam"`
	OSName             string                      `yaml:"os_name"`
	OSArch             string                      `yaml:"os_arch"`
	InstanceType       string                      `yaml:"instance_type"`
	WebserverPort      *int                        `yaml:"webserver_port,omitempty"`
	Properties         *string                     `yaml:"properties,omitempty"`
	Dirs               *map[string]string          `yaml:"dirs,omitempty"`
	Files              *map[string]string          `yaml:"files,omitempty"`
	ServiceInfo        *service.Service            `yaml:",omitempty"`
	Roles              map[string]ansible.Role     `yaml:",omitempty"`
	SecurityGroupRules []ansible.SecurityGroupRule `yaml:",omitempty"`
	Hosts              map[string]string           `yaml:",omitempty"`
	Generated          bool                        `yaml:",omitempty"`
}
