package generators

import (
	"github.com/cantara/nerthus2/executors/ansible"
	"github.com/cantara/nerthus2/system"
)

func GenerateServiceProvisioningPlay(serv system.Service, nodeVars map[string]any) (pb ansible.Playbook) {
	pb = ansible.Playbook{
		Name:       serv.Name,
		Hosts:      "localhost",
		Connection: "local",
		Vars:       map[string]any{},
	}
	var done []string
	for _, dep := range []string{
		"cron",
	} {
		addTask(dep, &pb, &done, serv.Roles)
	}
	addVars(serv.Vars, pb.Vars)
	addVars(nodeVars, pb.Vars)
	return
}
