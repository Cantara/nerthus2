package config

import (
	"embed"
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
		for _, cluster := range system.Clusters {
			for _, service := range cluster.Services {
				if service.ServiceInfo == nil {
					t.Fatal("serviceInfo was nil")
				}
			}
		}
	}
}
