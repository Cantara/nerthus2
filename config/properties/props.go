package properties

import (
	"errors"
	"fmt"
	"github.com/cantara/nerthus2/system"
	"strings"
)

type BootstrapVars struct {
	GitToken string
	GitRepo  string
	EnvName  string
}

func Calculate(serv system.Service, bootstrap *BootstrapVars) (propertiesName, properties string, err error) {
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
	if bootstrap != nil {
		properties = fmt.Sprintf("%s=%s\n%s=%s\n%s=%s\n%s", "git.token", bootstrap.GitToken, "git.repo", bootstrap.GitRepo, "boot_env", bootstrap.EnvName, properties)
	}
	propertiesName = serv.ServiceInfo.Requirements.PropertiesName
	return
}

var ErrHasWebserverPortAndNoKey = errors.New("webserver port and properties file provided without providing webserver_port_key")
