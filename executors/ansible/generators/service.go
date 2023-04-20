package generators

import (
	"github.com/cantara/nerthus2/executors/ansible"
	"github.com/cantara/nerthus2/system"
)

func GenerateServicePlay(cluster system.Cluster, serv system.Service, nodeVars map[string]any) (pb ansible.Playbook) {
	pb = ansible.Playbook{
		Name:       cluster.Name,
		Hosts:      "localhost",
		Connection: "local",
		Vars:       map[string]any{},
	}
	overrides := make([]string, len(cluster.Override))
	oi := 0
	for k := range cluster.Override {
		overrides[oi] = k
		oi++
	}
	var done []string
	for _, dep := range serv.ServiceInfo.Requirements.Roles {
		if arrayContains(overrides, dep) {
			continue
		}
		addTask(dep, &pb, &done, serv.Roles)
	}
	addVars(nodeVars, pb.Vars)
	return
}
