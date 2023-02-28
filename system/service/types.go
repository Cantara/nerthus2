package service

type Service struct {
	Name         string       `yaml:"name"`
	ServiceType  string       `yaml:"service_type"`
	HealthType   string       `yaml:"health_type"`
	Dependencies []string     `yaml:"dependencies"`
	Requirements Requirements `yaml:"requirements"`
}
type Requirements struct {
	RAM        string `yaml:"ram"`
	Disk       string `yaml:"disk"`
	CPU        int    `yaml:"cpu"`
	Properties string `yaml:"properties"`
}
