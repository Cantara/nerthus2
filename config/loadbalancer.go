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
		"fqdn":               env.FQDN,
		"loadbalancer_name":  sys.Loadbalancer,
		"loadbalancer_group": sys.LoadbalancerGroup,
		"cidr_base":          sys.CIDR,
		//"zone":               sys.Zone,
	}
	numberOfFrontendServices := 0
	var frontendTargetGroups []string
	for _, serv := range sys.Services {
		if !serv.ServiceInfo.Requirements.IsFrontend {
			continue
		}
		numberOfFrontendServices++
		frontendTargetGroups = append(frontendTargetGroups, serv.TargetGroup)
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
	for _, serv := range sys.Services {
		if serv.Playbook != "" {
			continue
		}
		if serv.ServiceInfo.Requirements.IsFrontend {
			continue
		}
		if serv.WebserverPort == nil {
			continue
		}
		if serv.TargetGroup == "" {
			continue
		}
		i++
		rules = append(rules, Rule{
			Conditions: []Condition{
				{
					Field: "path-pattern",
					Values: []string{
						fmt.Sprintf("/%s", serv.Name),
						fmt.Sprintf("/%s/*", serv.Name),
					},
				},
			},
			Actions: []Action{
				{
					TargetGroupName: serv.TargetGroup,
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
