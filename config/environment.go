package config

import (
	"embed"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config/systems"
	"github.com/cantara/nerthus2/system"
	"io/fs"
)

//go:embed builtin_roles
var builtinFS embed.FS

func ReadFullEnv(env string, baseFS fs.FS) (envConf system.Environment, err error) {
	builtinRoles, err := systems.BuiltinRoles(builtinFS)
	if err != nil {
		log.WithError(err).Fatal("while getting builtin roles")
	}
	envConf, err = systems.Environment(env, builtinRoles, baseFS)
	if err != nil {
		log.WithError(err).Fatal("while getting environment config")
	}
	for _, systemName := range envConf.Systems {
		systemConf, err := systems.System(envConf, systemName)
		if err != nil {
			log.WithError(err).Error("while getting system config")
			continue
		}
		envConf.SystemConfigs[systemName] = systemConf
		for _, cluster := range systemConf.Clusters {
			err = systems.Cluster(envConf, systemConf, cluster)
			if err != nil {
				log.WithError(err).Error("while getting service config")
				continue
			}
			/* FIXME? Missing logic for bootstrapping
			if bootstrap && strings.ToLower(cluster.ServiceInfo.Name) != "nerthus" {
				//log.Info("skipping service while bootstrap nerthus", "env", envConf.Name, "system", systemConf.Name, "service", service.Name)
				continue
			}
			*/
			//log.Info("executing service", "env", envConf.Name, "system", systemConf.Name, "service", service.Name, "overrides", service.Override)

		}
	}
	return
}
