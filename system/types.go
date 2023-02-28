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
	Name     string            `yaml:"name"`
	Vars     Vars              `yaml:"vars"`
	Expose   []int             `yaml:"expose,omitempty"`
	Playbook string            `yaml:"playbook,omitempty"`
	Git      string            `yaml:"git,omitempty"`
	Branch   string            `yaml:"branch,omitempty"`
	Override map[string]string `yaml:"override,omitempty"`
	Node     *ansible.Playbook `yaml:"-,omitempty"`
}
