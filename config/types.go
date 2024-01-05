package config

import (
	"strings"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config/schema"
)

type Environment struct {
	Name       string `json:"name"`
	NerthusURL string `json:"nerthus_url"`
	VisualeURL string `json:"visuale_url"`
	System     System `json:"system"`
	//Systems    []System `json:"systems"`
}
type Artifact struct {
	ID    string `json:"id"`
	Group string `json:"group"`
}
type Feature struct {
	Name     string `json:"name,omitempty"`
	Friendly string `json:"friendly,omitempty"`
	Tasks    []Task `json:"tasks,omitempty"`
}

func (f Feature) Privelaged() bool {
	for _, t := range f.Tasks {
		sbragi.Debug("privelaged", "type", t.Type, "install?", strings.HasPrefix(t.Type, "install"), "enable?", t.Type == "enable")
		if strings.HasPrefix(t.Type, "install") {
			return true
		}
		if t.Type == "enable" {
			return true
		}
	}
	return false
}

type Task struct {
	Info     string   `json:"info,omitempty"`
	Type     string   `json:"type,omitempty"`
	Source   string   `json:"source,omitempty"`
	Dest     string   `json:"dest,omitempty"`
	File     string   `json:"file,omitempty"`
	Url      string   `json:"url,omitempty"`
	Manager  string   `json:"manager,omitempty"`
	Package  *Package `json:"package,omitempty"`
	Service  string   `json:"service,omitempty"`
	Start    bool     `json:"start,omitempty"`
	Text     string   `json:"text,omitempty"`
	Data     []byte   `json:"data,omitempty"`
	Command  []string `json:"command,omitempty"`
	Username string   `json:"username,omitempty"`
	Root     bool     `json:"root"`
	//Vars    map[string]string
}

type Requirements struct {
	RAM              schema.ByteSize `json:"ram"`
	Disk             schema.ByteSize `json:"disk"`
	CPU              int             `json:"cpu"`
	PropertiesName   string          `json:"properties_name"`
	WebserverPortKey string          `json:"webserver_port_key"`
	NotClusterAble   bool            `json:"not_cluster_able"`
	IsFrontend       bool            `json:"is_frontend"`
	Features         []Feature       `json:"features,omitempty"`
	Packages         []Package       `json:"packages,omitempty"`
	Services         []string        `json:"services,omitempty"`
}
type Package struct {
	Name     string   `json:"name,omitempty"`
	Managers []string `json:"managers,omitempty"`
	//Provides []string `json:"~,omitempty"`
}

type ServiceInfo struct {
	Name         string       `json:"name"`
	ServiceType  string       `json:"service_type"`
	HealthType   string       `json:"health_type"`
	APIPath      string       `json:"api_path"`
	Artifact     Artifact     `json:"artifact"`
	Requirements Requirements `json:"requirements"`
}
type Service struct {
	Name        string      `json:"name"`
	MachineName string      `json:"machine_name"`
	Props       string      `json:"props"`
	Port        int         `json:"port"`
	Definition  ServiceInfo `json:"definition"`
}
type Cluster struct {
	Name     string                    `json:"name"`
	Node     Node                      `json:"node"`
	Services []Service                 `json:"services"`
	Size     int                       `json:"size"`
	Internal bool                      `json:"internal"`
	Packages map[string]schema.Package `json:"-"`
	System   []Feature                 `json:"system,omitempty"`
}

func (c Cluster) HasFrontend() bool {
	for _, serv := range c.Services {
		if serv.Definition.Requirements.IsFrontend {
			return true
		}
	}
	return false
}

func (c Cluster) IsClusterAble() bool {
	for _, serv := range c.Services {
		if serv.Definition.Requirements.NotClusterAble {
			return false
		}
	}
	return true
}

func (c Cluster) DiskSize() (size int) {
	size = 30
	for _, serv := range c.Services {
		size += serv.Definition.Requirements.Disk.ToGB()
	}
	return
}

type Node struct {
	Os   OS          `json:"os"`
	Arch schema.Arch `json:"arch"`
	Size string      `json:"size"`
}
type System struct {
	Name          string               `json:"name"`
	Domain        string               `json:"domain"`
	RoutingMethod schema.RoutingMethod `json:"routing_method"`
	Cidr          string               `json:"cidr"`
	Zone          string               `json:"zone"`
	Clusters      []Cluster            `json:"clusters"`
}
type PackageManager struct {
	Name   string   `json:"name,omitempty"`
	Syntax []string `json:"syntax,omitempty"`
	Local  []string `json:"local,omitempty"`
	Root   bool     `json:"root"`
}
type OS struct {
	Name            string
	PackageManagers []PackageManager `json:"package_managers,omitempty"`
	Provides        []string         `json:"provides,omitempty"`
}

func Contains[T comparable](arr []T, v T) int {
	for i, el := range arr {
		if el != v {
			continue
		}
		return i
	}
	return -1
}
