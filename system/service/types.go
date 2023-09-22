package service

type Service struct {
	Name         string       `yaml:"name"`
	ServiceType  string       `yaml:"service_type"`
	HealthType   string       `yaml:"health_type"`
	APIPath      string       `yaml:"api_path"`
	Artifact     Artifact     `yaml:"artifact"`
	Requirements Requirements `yaml:"requirements"`
}
type Requirements struct {
	RAM              string   `yaml:"ram"`
	Disk             string   `yaml:"disk"`
	CPU              int      `yaml:"cpu"`
	PropertiesName   string   `yaml:"properties_name"`
	WebserverPortKey string   `yaml:"webserver_port_key"`
	NotClusterAble   bool     `yaml:"not_cluster_able"`
	IsFrontend       bool     `yaml:"is_frontend"`
	Roles            []string `yaml:"roles"`
	Services         []string `yaml:"services"`
}
type Artifact struct {
	Id       string `yaml:"id"`
	Group    string `yaml:"group"`
	Release  string `yaml:"release"`
	Snapshot string `yaml:"snapshot"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}
