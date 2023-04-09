package generators

import (
	"github.com/cantara/nerthus2/executors/ansible"
	"github.com/cantara/nerthus2/system"
)

func GenerateNodePlay(serv system.Service, nodeVars map[string]any) (pb ansible.Playbook) {
	pb = ansible.Playbook{
		Name:       serv.Name,
		Hosts:      "localhost",
		Connection: "local",
		Vars:       map[string]any{},
	}
	overrides := make([]string, len(serv.Override))
	oi := 0
	for k := range serv.Override {
		overrides[oi] = k
		oi++
	}
	var done []string
	for _, dep := range serv.ServiceInfo.Dependencies {
		if arrayContains(overrides, dep) {
			continue
		}
		addTask(dep, &pb, &done, serv.Roles)
	}
	addTask("cron", &pb, &done, serv.Roles)
	addVars(serv.Vars, pb.Vars)
	addVars(nodeVars, pb.Vars)
	return
}
