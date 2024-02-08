package config

import (
	"github.com/cantara/nerthus2/config"
	"github.com/cantara/nerthus2/config/schema"
)

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
