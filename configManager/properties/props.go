package properties

/*
func Calculate(serv *system.Service) {
	if serv.Properties == nil {
		return
	}
	if serv.WebserverPort != nil {
		serv.Node.Vars["is_frontend"] = serv.ServiceInfo.Requirements.IsFrontend
		if serv.ServiceInfo.Requirements.WebserverPortKey == "" {
			log.Fatal("Webserver port and properties file provided without providing webserver_port_key")
		}
		lines := strings.Split(*serv.Properties, "\n")
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
			*serv.Properties = strings.Join(lines, "\n")
		} else {
			*serv.Properties = fmt.Sprintf("%s=%d\n%s", serv.ServiceInfo.Requirements.WebserverPortKey, *serv.WebserverPort, *serv.Properties)
		}
	}
}
*/
