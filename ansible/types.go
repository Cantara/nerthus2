package ansible

type Playbook struct {
	Name       string            `yaml:"name"`
	Hosts      string            `yaml:"hosts"`
	Connection string            `yaml:"connection"`
	Vars       map[string]string `yaml:"vars"`
	Tasks      []map[string]any  `yaml:"tasks"`
}

type Role struct {
	Id           string
	Name         string            `yaml:"name"`
	Dependencies []Dependency      `yaml:"dependencies"`
	Vars         map[string]string `yaml:"vars"`
	Tasks        []map[string]any  `yaml:"tasks"`
}

type Dependency struct {
	Role string `yaml:"role"`
}

type Tasks map[string]any

type SecurityGroupRule struct {
	Proto    string `yaml:"proto" json:"proto"`
	FromPort string `yaml:"from_port" json:"from_port"`
	ToPort   string `yaml:"to_port" json:"to_port"`
	Group    string `yaml:"group_name" json:"group_name"`
}
