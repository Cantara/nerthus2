package config

import (
	"github.com/cantara/nerthus2/system"
	"os"
	"strings"
)

func ClusterProvisioningVars(env system.Environment, sys system.System, clust system.Cluster, bootstrap bool) (vars map[string]any) {
	vars = map[string]any{
		"region":               os.Getenv("aws.region"),
		"env":                  env.Name,
		"nerthus_host":         env.Nerthus,
		"visuale_host":         env.Visuale,
		"system":               sys.Name,
		"service":              clust.Name,
		"name_base":            sys.Scope,
		"vpc_name":             sys.VPC,
		"key_name":             sys.Key,
		"node_names":           clust.NodeNames,
		"loadbalancer_name":    sys.Loadbalancer,
		"loadbalancer_group":   sys.LoadbalancerGroup,
		"security_group_name":  clust.SecurityGroup,
		"security_group_rules": clust.SecurityGroupRules,
		"is_frontend":          clust.IsClusterAble(),
		"os_name":              clust.OSName,
		"os_arch":              clust.OSArch,
		"instance_type":        clust.InstanceType,
		"cidr_base":            sys.CIDR,
		"zone":                 sys.Zone,
		"iam_profile":          clust.IAM,
		"cluster_name":         clust.ClusterName,
		"cluster_ports":        clust.Expose,
		"cluster_info":         clust.ClusterInfo,
	}
	if clust.HasWebserverPort() {
		vars["webserver_port"] = clust.GetWebserverPort()
	}
	if clust.TargetGroup != "" {
		vars["target_group_name"] = clust.TargetGroup
	}
	if bootstrap {
		boots := make([]string, len(clust.NodeNames))
		for i := 0; i < len(boots); i++ {
			boots[i] = `cat <<'EOF' > bootstrap.yml
{{ lookup('file', 'nodes/` + clust.NodeNames[i] + `_bootstrap.yml') }}
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
