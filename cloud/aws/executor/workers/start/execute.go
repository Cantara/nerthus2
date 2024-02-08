package start

import (
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/probe/config"
)

/*
type Start struct {
	Env        string               `json:"env"`
	System     string               `json:"system"`
	Cluster    string               `json:"cluster"`
	OSName     string               `json:"os_name"`
	Arch       ami.Arch             `json:"arch"`
	Network    string               `json:"network"`
	Nodes      []string             `json:"node_names"`
	Size       string               `json:"size"`
	DiskSize   int                  `json:"disk_size"`
	Nerthus    string               `json:"nerthus"`
	Visuale    string               `json:"visuale"`
	Path       string               `json:"path"`
	Port       int                  `json:"port"`
	Base       string               `json:"base"`
	Routing    system.RoutingMethod `json:"routing"`
	Domain     string               `json:"domain"`
	IsFrontend bool                 `json:"is_frontend"`
}
*/

/*
type Environment struct {
	Name        string `json:"name"`
	MachineName string `json:"machine_name"`
	NerthusURL  string `json:"nerthus_url"`
	VisualeURL  string `json:"visuale_url"`
	System      System `json:"system"`
}

type System struct {
	Name          string               `json:"name"`
	MachineName   string               `json:"machine_name"`
	Domain        string               `json:"domain"`
	RoutingMethod schema.RoutingMethod `json:"routing_method"`
	Cidr          string               `json:"cidr"`
	Zone          string               `json:"zone"`
	Cluster       config.Cluster       `json:"cluster"`
}
*/

var Fingerprint = adapter.New[config.Environment](adapter.Start)
var Adapter = Fingerprint.Adapter(func(a []adapter.Value) (s config.Environment, err error) {
	return Fingerprint.Value(a[0]), nil
}, Fingerprint)
