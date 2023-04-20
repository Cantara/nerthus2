package config

import (
	"embed"
	"github.com/cantara/nerthus2/executors/ansible/generators"
	"log"
	"os"
	"testing"
)

//go:embed test_config/systems
var baseFS embed.FS

func TestReadFullEnv(t *testing.T) {
	envConf, err := ReadFullEnv("test_env", os.DirFS("test_config")) //baseFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(envConf.Roles) == 0 {
		t.Fatal("env roles missing")
	}
	if len(envConf.SystemConfigs) < 4 {
		t.Fatal("missing system configs")
	}
	if envConf.Domain == "" {
		t.Fatal("env missing domain")
	}
	for _, system := range envConf.SystemConfigs {
		if system.Domain == "" {
			t.Fatal("system missing domain")
		}
		if len(system.Roles) == 0 {
			t.Fatal("system roles missing")
		}
		for _, cluster := range system.Clusters {
			if len(cluster.Roles) == 0 {
				t.Fatal("cluster roles missing")
			}
			for _, service := range cluster.Services {
				log.Println("verifying", "env", envConf.Name, "system", system.Name, "cluster", cluster.Name, "service", service.Name)
				if service.ServiceInfo == nil {
					t.Fatal("serviceInfo was nil")
				}
				serviceVars := ServiceProvisioningVars(envConf, system, *cluster, *service)
				serviceNodeVars := ServiceNodeVars(*cluster, 0, serviceVars)
				servicePlay := generators.GenerateServicePlay(*cluster, *service, serviceNodeVars)
				if len(servicePlay.Tasks) == 0 {
					t.Fatal("tasks missing in play")
				}
				if service.Name == "nerthus" {
					if !service.ServiceInfo.Requirements.IsFrontend {
						t.Fatal("nerthus is not frontend")
					}
					if servicePlay.Vars["is_frontend"] != true {
						t.Fatal("nerthus service play var is_frontend is not true")
					}
				}
				if cluster.Name == "visuale" {
					if !arrayContains(service.ServiceInfo.Requirements.Roles, "service_files") {
						t.Fatal("visuale was missing service_files role")
					}
					if _, ok := servicePlay.Vars["service"]; !ok {
						t.Fatal("service variable missing", "service", service.Name)
					}
				}
			}
		}
	}
}
