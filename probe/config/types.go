package config

import "github.com/cantara/nerthus2/config"

type Environment struct {
	Name        string `json:"name"`
	MachineName string `json:"machine_name"`
	NerthusURL  string `json:"nerthus_url"`
	VisualeURL  string `json:"visuale_url"`
	System      System `json:"system"`
}
type System struct {
	Name          string  `json:"name"`
	MachineName   string  `json:"machine_name"`
	Domain        string  `json:"domain"`
	RoutingMethod string  `json:"routing_method"`
	Cidr          string  `json:"cidr"`
	Zone          string  `json:"zone"`
	Cluster       Cluster `json:"cluster"`
}
type Cluster struct {
	Name        string                    `json:"name"`
	MachineName string                    `json:"machine_name"`
	Node        config.Node               `json:"node"`
	Services    []config.Service          `json:"services"`
	Internal    bool                      `json:"internal"`
	Packages    map[string]config.Package `json:"-"`
	System      []config.Feature          `json:"system,omitempty"`
}
