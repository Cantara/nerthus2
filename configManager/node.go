package configManager

import (
	"errors"
	"fmt"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/configManager/file"
	"github.com/cantara/nerthus2/configManager/file/dirReader"
	"github.com/cantara/nerthus2/system"
	"io/fs"
	"strconv"
	"strings"
)

var ErrHasWebserverPortAndNoKey = errors.New("webserver port and properties file provided without providing webserver_port_key")

func GenerateProperties(serv system.Service) (propertiesName, properties string, err error) {
	if serv.Properties != nil {
		properties = *serv.Properties
	}
	if serv.WebserverPort != nil {
		if serv.ServiceInfo.Requirements.WebserverPortKey == "" {
			err = ErrHasWebserverPortAndNoKey
			return
		}
		lines := strings.Split(properties, "\n")
		found := false
		for l, line := range lines {
			if !strings.HasPrefix(line, serv.ServiceInfo.Requirements.WebserverPortKey) {
				continue
			}
			lines[l] = fmt.Sprintf("%s=%d", serv.ServiceInfo.Requirements.WebserverPortKey, *serv.WebserverPort)
			found = true
			break
		}
		if found {
			properties = strings.Join(lines, "\n")
		} else {
			properties = fmt.Sprintf("%s=%d\n%s", serv.ServiceInfo.Requirements.WebserverPortKey, *serv.WebserverPort, properties)
		}
	}
	propertiesName = serv.ServiceInfo.Requirements.PropertiesName
	return
}

func GenerateNodeVars(envFS fs.FS, configDir string, serv system.Service, nodeName string, nodeNum int) (vars map[string]any, err error) {
	vars["hostname"] = nodeName
	vars["server_number"] = strconv.Itoa(nodeNum)

	if serv.Properties != nil { //This is now slightly less stupid
		propertiesName, properties, err := GenerateProperties(serv)
		if err != nil {
			return nil, err
		}
		vars["properties_name"] = propertiesName
		vars["local_override_content"] = properties
	}
	var allFiles []file.File
	if serv.Dirs != nil {
		for localDir, nodeDir := range *serv.Dirs {
			files, err := dirReader.ReadFilesFromDir(envFS, configDir, localDir, nodeDir)
			if err != nil {
				log.WithError(err).Error("while reading files from disk", "env", envFS, "config", configDir, "local", localDir, "node", nodeDir)
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
	vars["artifact_group"] = serv.ServiceInfo.Artifact.Group
	if serv.ServiceInfo.Artifact.Release != "" {
		vars["artifact_release"] = serv.ServiceInfo.Artifact.Release
	}
	return
}

/*
func MarshalPlaybook(vars map[string]any, playbook ansible.Playbook) (out []byte, err error) {
	//vars, err := GenerateNodeVars(envFS, configDir, serv, nodeName, nodeNum)
	playbook.Vars = vars
	out, err = yaml.Marshal([]ansible.Playbook{
		playbook,
	})
	if err != nil {
		return
		//log.WithError(err).Fatal("unable to marshall yaml for node playbook", "node", serv.Node)
	}
	return
}

func GenerateNodeProvisionPlay(serv system.Service, name string, i int) (out []byte, err error) {
	serv.Prov.Vars["hostname"] = name
	serv.Prov.Vars["server_number"] = strconv.Itoa(i)

	out, err = yaml.Marshal([]ansible.Playbook{
		*serv.Prov,
	})
	if err != nil {
		log.WithError(err).Fatal("unable to marshall yaml for node playbook", "node", serv.Prov)
	}
	return
}
*/
