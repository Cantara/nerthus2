package generators

import (
	"github.com/cantara/nerthus2/executors/ansible"
	"github.com/cantara/nerthus2/system"
)

func GenerateNodeProvisioningPlay(cluster system.Cluster, nodeVars map[string]any) (pb ansible.Playbook) {
	pb = ansible.Playbook{
		Name:       cluster.Name,
		Hosts:      "localhost",
		Connection: "local",
		Vars:       map[string]any{},
	}
	var done []string
	for _, dep := range []string{
		"cron",
	} {
		addTask(dep, &pb, &done, cluster.Roles)
	}
	addVars(nodeVars, pb.Vars)
	return
}
