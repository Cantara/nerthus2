package config

import (
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config/properties"
	"github.com/cantara/nerthus2/config/readers/dir"
	"github.com/cantara/nerthus2/config/readers/file"
	"github.com/cantara/nerthus2/system"
	"strconv"
)

func NodeProvisioningVars(serv system.Service, nodeNum int, systemProvisioningVars map[string]any) (vars map[string]any) {
	vars = map[string]any{}
	addVars(systemProvisioningVars, vars)
	delete(vars, "bootstrap")
	delete(vars, "security_group_rules")

	vars["hostname"] = serv.NodeNames[nodeNum]
	vars["server_number"] = strconv.Itoa(nodeNum)
	vars["service"] = "ec2-user"

	return
}

type BootstrapVars struct {
	GitToken string
	GitRepo  string
	EnvName  string
}

func NodeBootstrapVars(env system.Environment, sys system.System, serv system.Service, nodeNum int, serviceProvisioningVars map[string]any, bootstrap *BootstrapVars) (vars map[string]any) {
	vars = map[string]any{}
	addVars(serviceProvisioningVars, vars)
	delete(vars, "bootstrap")
	delete(vars, "security_group_rules")
	addVars(env.Vars, vars)
	addVars(sys.Vars, vars)
	addVars(serv.Vars, vars)

	if bootstrap != nil {
		vars["git_token"] = bootstrap.GitToken
		vars["git_repo"] = bootstrap.GitRepo
		vars["boot_env"] = bootstrap.EnvName
	}

	vars["hostname"] = serv.NodeNames[nodeNum]
	vars["server_number"] = strconv.Itoa(nodeNum)
	if serv.Properties != nil {
		propertiesName, props, err := properties.Calculate(serv)
		if err != nil {
			log.WithError(err).Fatal("temptest")
			return
		}
		vars["properties_name"] = propertiesName
		vars["local_override_content"] = props
	}
	var allFiles []file.File
	if serv.Dirs != nil {
		for localDir, nodeDir := range *serv.Dirs {
			files, err := dir.ReadFilesFromDir(sys.FS, localDir, nodeDir)
			if err != nil {
				log.WithError(err).Error("while reading files from disk", "sys", sys.FS, "local", localDir, "node", nodeDir)
				continue
			}
			if len(allFiles) == 0 {
				allFiles = files
				continue
			}
			allFiles = append(allFiles, files...)
		}
	}
	func() {
		if serv.Files != nil {
			files := file.FilesFromConfig(*serv.Files)
			if len(allFiles) == 0 {
				allFiles = files
				return
			}
			allFiles = append(allFiles, files...)
		}
	}()
	if len(allFiles) > 0 {
		vars["files"] = allFiles
	}

	vars["health_type"] = serv.ServiceInfo.HealthType
	vars["artifact_id"] = serv.ServiceInfo.Artifact.Id
	vars["artifact_group"] = serv.ServiceInfo.Artifact.Group
	vars["artifact_release"] = serv.ServiceInfo.Artifact.Release
	vars["artifact_snapshot"] = serv.ServiceInfo.Artifact.Snapshot
	vars["artifact_user"] = serv.ServiceInfo.Artifact.User
	vars["artifact_password"] = serv.ServiceInfo.Artifact.Password
	vars["service_type"] = serv.ServiceInfo.ServiceType
	return
}
