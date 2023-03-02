package system

import "github.com/cantara/nerthus2/ansible"

type System struct {
	Name     string    `yaml:"name"`
	Env      string    `yaml:"env"`
	Vars     Vars      `yaml:"vars"`
	Services []Service `yaml:"services"`
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
	Node          *ansible.Playbook `yaml:"-,omitempty"`
}
