package config

import (
	"fmt"
	"github.com/cantara/nerthus2/system"
	"os"
	"strings"
)

type Condition struct {
	Field  string   `json:"Field"`
	Values []string `json:"Values"`
}
type Action struct {
	TargetGroupName string `json:"TargetGroupName"`
	Type            string `json:"Type"`
}

type Rule struct {
	Conditions []Condition `json:"Conditions"`
	Actions    []Action    `json:"Actions"`
	Priority   int         `json:"Priority"`
}

func SystemLoadbalancerVars(env system.Environment, sys system.System) (vars map[string]any) {
	vars = map[string]any{
		"region":             os.Getenv("aws.region"),
		"env":                env.Name,
		"system":             sys.Name,
		"name_base":          sys.Scope,
		"vpc_name":           sys.VPC,
		"key_name":           sys.Key,
		"fqdn":               fmt.Sprintf("*.%s", sys.Domain),
		"loadbalancer_name":  sys.Loadbalancer,
		"loadbalancer_group": sys.LoadbalancerGroup,
		"cidr_base":          sys.CIDR,
		//"zone":               sys.Zone,
	}
	numberOfFrontendServices := 0
	var frontendTargetGroups []string
	for _, cluster := range sys.Clusters { // This whole thing seems weird
		for _, serv := range cluster.Services {
			if !serv.ServiceInfo.Requirements.IsFrontend {
				continue
			}
			numberOfFrontendServices++
			frontendTargetGroups = append(frontendTargetGroups, cluster.TargetGroup)
			break // This logic is forcing max one frontend per cluster. This seems like a weird solution
		}
	}
	if numberOfFrontendServices == 1 {
		vars["default_actions"] = []Action{
			{
				TargetGroupName: frontendTargetGroups[0],
				Type:            "forward",
			},
		}
	}

	i := 0
	rules := []Rule{}
	for _, cluster := range sys.Clusters {
		if cluster.Playbook != "" {
			continue
		}
		if cluster.HasFrontend() { //TODO: This should also add a part about Routing method
			continue
		}
		if !cluster.HasWebserverPort() {
			continue
		}
		if cluster.TargetGroup == "" { //This seems redundant
			continue
		}
		i++
		var cond Condition
		switch sys.RoutingMethod {
		case system.RoutingPath:
			cond = Condition{
				Field: "path-pattern",
				Values: []string{
					fmt.Sprintf("/%s", cluster.Name),
					fmt.Sprintf("/%s/*", cluster.Name),
				},
			}
		case system.RoutingHost:
			cond = Condition{
				Field: "host-header",
				Values: []string{
					fmt.Sprintf("%s-%s.%s", sys.Name, cluster.Name, env.Domain),
				},
			}
		}
		rules = append(rules, Rule{
			Conditions: []Condition{cond},
			Actions: []Action{
				{
					TargetGroupName: cluster.TargetGroup,
					Type:            "forward",
				},
			},
			Priority: i,
		})
	}
	vars["rules"] = rules
	if strings.ToLower(os.Getenv("allowAllRegions")) == "true" {
		if r, ok := sys.Vars["region"]; ok && r != "" {
			vars["region"] = r
		} else if r, ok = env.Vars["region"]; ok && r != "" {
			vars["region"] = r
		}
	}
	return
}
