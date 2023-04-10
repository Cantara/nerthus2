package generators

import (
	"fmt"
	"github.com/cantara/nerthus2/executors/ansible"
	"gopkg.in/yaml.v2"
)

func PlayToYaml(pb ansible.Playbook) (play []byte, err error) {
	return yaml.Marshal([]ansible.Playbook{
		pb,
	})
}

func addTask(role string, pb *ansible.Playbook, done *[]string, roles map[string]ansible.Role) {
	if arrayContains(*done, role) {
		return
	}
	r, ok := roles[role]
	if !ok {
		return
	}
	for _, req := range r.Dependencies {
		addTask(req.Role, pb, done, roles)
	}
	addVars(r.Vars, pb.Vars)
	pb.Tasks = append(pb.Tasks, r.Tasks...)
	*done = append(*done, r.Id)
}

func addVars[T comparable](inVars map[string]T, outVars map[string]any) {
	for k, v := range inVars {
		//var zero T
		if fmt.Sprint(v) == "" { //v == zero { //Excluding all zero values might not be optimal for items like ints.
			continue
		}
		outVars[k] = v
	}
}

func arrayContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
