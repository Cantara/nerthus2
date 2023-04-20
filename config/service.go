package config

import (
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config/properties"
	"github.com/cantara/nerthus2/config/readers/dir"
	"github.com/cantara/nerthus2/config/readers/file"
	"github.com/cantara/nerthus2/system"
	"os"
	"strconv"
	"strings"
)

func ServiceProvisioningVars(env system.Environment, sys system.System, cluster system.Cluster, serv system.Service) (vars map[string]any) {
	vars = map[string]any{}
	addVars(env.Vars, vars)
	addVars(sys.Vars, vars)
	addVars(cluster.Vars, vars)
	addVars(map[string]any{
		"region":              os.Getenv("aws.region"),
		"env":                 env.Name,
		"domain":              sys.Domain,
		"nerthus_host":        env.Nerthus,
		"visuale_host":        env.Visuale,
		"system":              sys.Name,
		"cluster":             cluster.Name,
		"service":             serv.Name,
		"name_base":           sys.Scope,
		"vpc_name":            sys.VPC,
		"key_name":            sys.Key,
		"node_names":          cluster.NodeNames,
		"loadbalancer_name":   sys.Loadbalancer,
		"security_group_name": cluster.SecurityGroup,
		"cidr_base":           sys.CIDR,
		"zone":                sys.Zone,
		"cluster_name":        cluster.ClusterName,
		"cluster_ports":       cluster.Expose,
		"cluster_info":        cluster.ClusterInfo,
	}, vars)
	if cluster.HasWebserverPort() {
		vars["webserver_port"] = cluster.GetWebserverPort()
	}

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
		vars["dirs"] = file.DirsForFiles(allFiles)
	}

	vars["health_type"] = serv.ServiceInfo.HealthType
	vars["artifact_id"] = serv.ServiceInfo.Artifact.Id
	vars["artifact_group"] = serv.ServiceInfo.Artifact.Group
	vars["artifact_release"] = serv.ServiceInfo.Artifact.Release
	vars["artifact_snapshot"] = serv.ServiceInfo.Artifact.Snapshot
	vars["artifact_user"] = serv.ServiceInfo.Artifact.User
	vars["artifact_password"] = serv.ServiceInfo.Artifact.Password
	vars["service_type"] = serv.ServiceInfo.ServiceType
	vars["is_frontend"] = serv.ServiceInfo.Requirements.IsFrontend

	if strings.ToLower(os.Getenv("allowAllRegions")) == "true" {
		if r, ok := sys.Vars["region"]; ok && r != "" {
			vars["region"] = r
		} else if r, ok = env.Vars["region"]; ok && r != "" {
			vars["region"] = r
		}
	}
	return
}

func ServiceNodeVars(cluster system.Cluster, nodeNum int, vars map[string]any) (outVars map[string]any) {
	outVars = map[string]any{}
	addVars(vars, outVars)
	outVars["hostname"] = cluster.NodeNames[nodeNum]
	outVars["server_number"] = strconv.Itoa(nodeNum)
	return
}

func NodeProvisioningVars(cluster system.Cluster, nodeNum int, vars map[string]any) (outVars map[string]any) {
	outVars = map[string]any{}
	addVars(vars, outVars)
	outVars["hostname"] = cluster.NodeNames[nodeNum]
	outVars["server_number"] = strconv.Itoa(nodeNum)
	outVars["service"] = "ec2-user"
	return
}
