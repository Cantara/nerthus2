package config

import (
	"github.com/cantara/nerthus2/system"
	"os"
	"strings"
)

func ClusterProvisioningVars(env system.Environment, sys system.System, cluster system.Cluster, bootstrap bool) (vars map[string]any) {
	vars = map[string]any{}
	addVars(env.Vars, vars)
	addVars(sys.Vars, vars)
	addVars(cluster.Vars, vars)
	addVars(map[string]any{
		"region":               os.Getenv("aws.region"),
		"env":                  env.Name,
		"nerthus_host":         env.Nerthus,
		"visuale_host":         env.Visuale,
		"system":               sys.Name,
		"service":              cluster.Name,
		"name_base":            sys.Scope,
		"vpc_name":             sys.VPC,
		"key_name":             sys.Key,
		"node_names":           cluster.NodeNames,
		"loadbalancer_name":    sys.Loadbalancer,
		"loadbalancer_group":   sys.LoadbalancerGroup,
		"security_group_name":  cluster.SecurityGroup,
		"security_group_rules": cluster.SecurityGroupRules,
		"is_frontend":          cluster.HasFrontend(),
		"os_name":              cluster.OSName,
		"os_arch":              cluster.OSArch,
		"instance_type":        cluster.InstanceType,
		"cidr_base":            sys.CIDR,
		"zone":                 sys.Zone,
		"iam_profile":          cluster.IAM,
		"cluster_name":         cluster.ClusterName,
		"cluster_ports":        cluster.Expose,
		"cluster_info":         cluster.ClusterInfo,
		"routing_method":       sys.RoutingMethod,
	}, vars)
	if cluster.HasWebserverPort() {
		vars["webserver_port"] = cluster.GetWebserverPort()
	}
	if cluster.TargetGroup != "" {
		vars["target_group_name"] = cluster.TargetGroup
	}
	if bootstrap {
		boots := make([]string, len(cluster.NodeNames))
		for i := 0; i < len(boots); i++ {
			boots[i] = `cat <<'EOF' > bootstrap.yml
{{ lookup('file', 'nodes/` + cluster.NodeNames[i] + `_bootstrap.yml') }}
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
