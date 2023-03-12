package system

import (
	"github.com/cantara/nerthus2/ansible"
	"github.com/cantara/nerthus2/system/service"
)

type Environment struct {
	Name    string   `yaml:"name"`
	Vars    Vars     `yaml:"vars"`
	Systems []string `yaml:"systems"`
}

type System struct {
	Name              string    `yaml:"name"`
	Vars              Vars      `yaml:"vars"`
	Services          []Service `yaml:"services"`
	Scope             string    `yaml:"scope"`
	VPC               string    `yaml:"vpc"`
	Key               string    `yaml:"key"`
	Loadbalancer      string    `yaml:"loadbalancer"`
	LoadbalancerGroup string    `yaml:"loadbalancerGroup"`
}

type Vars map[string]any

type Service struct {
	Name          string            `yaml:"name"`
	Vars          Vars              `yaml:"vars"`
	Expose        []int             `yaml:"expose,omitempty"`
	Playbook      string            `yaml:"playbook,omitempty"`
	Local         string            `yaml:"local,omitempty"`
	Git           string            `yaml:"git,omitempty"`
	Branch        string            `yaml:"branch,omitempty"`
	Override      map[string]string `yaml:"override,omitempty"`
	Internal      bool              `yaml:"internal"`
	NumberOfNodes int               `yaml:"number_of_nodes"`
	NodeNames     []string          `yaml:"node_names"`
	SecurityGroup string            `yaml:"security_group"`
	TargetGroup   string            `yaml:"target_group"`
	WebserverPort *int              `yaml:"webserver_port,omitempty"`
	Node          *ansible.Playbook `yaml:",omitempty"`
	ServiceInfo   *service.Service  `yaml:",omitempty"`
}
