package config

import (
	"github.com/cantara/nerthus2/system"
	"os"
	"strings"
)

func ServiceProvisioningVars(env system.Environment, sys system.System, serv system.Service, bootstrap bool) (vars map[string]any) {
	vars = map[string]any{
		"region":               os.Getenv("aws.region"),
		"env":                  env.Name,
		"nerthus_host":         env.Nerthus,
		"visuale_host":         env.Visuale,
		"system":               sys.Name,
		"service":              serv.Name,
		"name_base":            sys.Scope,
		"vpc_name":             sys.VPC,
		"key_name":             sys.Key,
		"node_names":           serv.NodeNames,
		"loadbalancer_name":    sys.Loadbalancer,
		"loadbalancer_group":   sys.LoadbalancerGroup,
		"security_group_name":  serv.SecurityGroup,
		"security_group_rules": serv.SecurityGroupRules,
		"is_frontend":          serv.ServiceInfo.Requirements.IsFrontend,
		"os_name":              serv.OSName,
		"os_arch":              serv.OSArch,
		"instance_type":        serv.InstanceType,
		"cidr_base":            sys.CIDR,
		"zone":                 sys.Zone,
		"iam_profile":          serv.IAM,
		"cluster_name":         serv.ClusterName,
		"cluster_ports":        serv.Expose,
		"cluster_info":         serv.ClusterInfo,
	}
	if serv.WebserverPort != nil {
		vars["webserver_port"] = serv.WebserverPort
	}
	if serv.TargetGroup != "" {
		vars["target_group_name"] = serv.TargetGroup
	}
	if bootstrap {
		boots := make([]string, len(serv.NodeNames))
		for i := 0; i < len(boots); i++ {
			boots[i] = `cat <<'EOF' > bootstrap.yml
{{ lookup('file', 'nodes/` + serv.NodeNames[i] + `_bootstrap.yml') }}
EOF
su -c "ansible-playbook bootstrap.yml" ec2-user`
		}
		vars["bootstrap"] = boots
	}
	if strings.ToLower(os.Getenv("allowAllRegions")) == "true" {
		if r, ok := sys.Vars["region"]; ok && r != "" {
			vars["region"] = r
		} else if r, ok = env.Vars["region"]; ok && r != "" {
			vars["region"] = r
		}
	}
	return
}