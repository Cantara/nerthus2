package executors

import (
	"fmt"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/ansible"
	"github.com/cantara/nerthus2/configManager"
	"github.com/cantara/nerthus2/configManager/file"
	"github.com/cantara/nerthus2/configManager/file/dirReader"
	"github.com/cantara/nerthus2/system"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"strings"
)

func WriteNodePlay(nodeDir, nodeName string, play []byte, bootstrap bool) (err error) {
	os.Mkdir(nodeDir, 0750)
	if bootstrap {
		nodeName = fmt.Sprintf("%s_bootstrap", nodeName)
	}
	fn := fmt.Sprintf("%s/%s.yml", nodeDir, nodeName)
	os.Remove(fn)
	err = os.WriteFile(fn, play, 0640)
	if err != nil {
		return
	}
	return
}

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

func PlayToYaml(pb ansible.Playbook) (play []byte, err error) {
	return yaml.Marshal([]ansible.Playbook{
		pb,
	})
}

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

func NodeProvisioningVars(serv system.Service, nodeNum int, systemProvisioningVars map[string]any) (vars map[string]any) {
	vars = map[string]any{}
	addVars(systemProvisioningVars, vars)
	delete(vars, "bootstrap")
	delete(vars, "security_group_rules")

	vars["hostname"] = serv.NodeNames[nodeNum]
	vars["server_number"] = strconv.Itoa(nodeNum)
	vars["service"] = "ec2-user"

	return
}

type BootstrapVars struct {
	GitToken string
	GitRepo  string
	EnvName  string
}

func NodeBootstrapVars(env system.Environment, sys system.System, serv system.Service, nodeNum int, serviceProvisioningVars map[string]any, bootstrap *BootstrapVars) (vars map[string]any) {
	vars = map[string]any{}
	addVars(serviceProvisioningVars, vars)
	delete(vars, "bootstrap")
	delete(vars, "security_group_rules")
	addVars(env.Vars, vars)
	addVars(sys.Vars, vars)
	addVars(serv.Vars, vars)

	if bootstrap != nil {
		vars["git_token"] = bootstrap.GitToken
		vars["git_repo"] = bootstrap.GitRepo
		vars["boot_env"] = bootstrap.EnvName
	}

	vars["hostname"] = serv.NodeNames[nodeNum]
	vars["server_number"] = strconv.Itoa(nodeNum)
	if serv.Properties != nil {
		propertiesName, properties, err := configManager.GenerateProperties(serv)
		if err != nil {
			log.WithError(err).Fatal("temptest")
			return
		}
		//vars["properties_name"] = serv.ServiceInfo.Requirements.PropertiesName
		//vars["local_override_content"] = *serv.Properties
		vars["properties_name"] = propertiesName
		vars["local_override_content"] = properties
	}
	var allFiles []file.File
	if serv.Dirs != nil {
		for localDir, nodeDir := range *serv.Dirs {
			files, err := dirReader.ReadFilesFromDir(sys.FS, localDir, nodeDir)
			if err != nil {
				log.WithError(err).Error("while reading files from disk", "sys", sys.FS, "local", localDir, "node", nodeDir)
				continue
			}
			if len(allFiles) == 0 {
				allFiles = files
				continue
			}
			allFiles = append(allFiles, files...)
		}
	}
	func() {
		if serv.Files != nil {
			files := file.FilesFromConfig(*serv.Files)
			if len(allFiles) == 0 {
				allFiles = files
				return
			}
			allFiles = append(allFiles, files...)
		}
	}()
	if len(allFiles) > 0 {
		vars["files"] = allFiles
	}

	vars["health_type"] = serv.ServiceInfo.HealthType
	vars["artifact_id"] = serv.ServiceInfo.Artifact.Id
	vars["artifact_group"] = serv.ServiceInfo.Artifact.Group
	vars["artifact_release"] = serv.ServiceInfo.Artifact.Release
	vars["artifact_snapshot"] = serv.ServiceInfo.Artifact.Snapshot
	vars["artifact_user"] = serv.ServiceInfo.Artifact.User
	vars["artifact_password"] = serv.ServiceInfo.Artifact.Password
	vars["service_type"] = serv.ServiceInfo.ServiceType

	//NODE FILE CONTAINS MORE OF THIS
	return
}

/*
serv.Node.Vars["artifact_group"] = serv.ServiceInfo.ArtifactGroup
if serv.ServiceInfo.ArtifactRelease != "" {
serv.Node.Vars["artifact_release"] = serv.ServiceInfo.ArtifactRelease
}
out, err = yaml.Marshal([]ansible.Playbook{
*serv.Node,
})
if err != nil {
log.WithError(err).Fatal("unable to marshall yaml for node playbook", "node", serv.Node)
}
return
}

	func GenerateNodeProvisionPlay(serv system.Service, name string, i int) (out []byte, err error) {
		serv.Prov.Vars["hostname"] = name
		serv.Prov.Vars["server_number"] = strconv.Itoa(i)

		out, err = yaml.Marshal([]ansible.Playbook{
			*serv.Prov,
		})
		if err != nil {
			log.WithError(err).Fatal("unable to marshall yaml for node playbook", "node", serv.Prov)
		}
		return
	}
*/

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
		"target_group_name":    serv.TargetGroup,
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
	}
	if serv.WebserverPort != nil {
		vars["webserver_port"] = serv.WebserverPort
	}
	for service, clusterName := range serv.Hosts {
		vars[fmt.Sprintf("%s_cluster_name", strings.ReplaceAll(service, "-", "_"))] = clusterName
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
	return
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
		"zone":               sys.Zone,
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
				TargetGroupName: sys.Services[0].TargetGroup,
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
	return
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

/*
	extraVars := map[string]any{}
	//addVars(serv.Node.Vars, extraVars)
	addVars(env.Vars, extraVars)
	addVars(sys.Vars, extraVars)
	addVars(serv.Vars, extraVars)
	addVars(nerthusVars, extraVars)

	var extra string
	if systemName != serv.Name {
		extra = fmt.Sprintf("-%s", serv.Name)
	}
	if serv.ServiceInfo.ArtifactId != "" {
		extraVars["artifact_id"] = serv.ServiceInfo.ArtifactId
	} else {
		extraVars["artifact_id"] = serv.Name
	}
	log.Info("vars", "key_name", extraVars["key_name"], "vpc_name", extraVars["vpc_name"])
	if len(serv.NodeNames) == 0 {
		if serv.NumberOfNodes == 1 {
			serv.NodeNames = []string{
				fmt.Sprintf("%s%s", sys.Scope, extra),
			}
		} else {
			serv.NodeNames = make([]string, serv.NumberOfNodes)
			for num := 1; num <= serv.NumberOfNodes; num++ {
				serv.NodeNames[num-1] = fmt.Sprintf("%s%s-%d", sys.Scope, extra, num)
			}
		}
	}
	if len(serv.NodeNames) != serv.NumberOfNodes {
		log.Fatal("provided node names does not match number of nodes", "numberOfNodes", serv.NumberOfNodes, "nodeNames", serv.NodeNames)
	}
	extraVars["node_names"] = serv.NodeNames
	if serv.SecurityGroup == "" {
		serv.SecurityGroup = fmt.Sprintf("%s%s-sg", sys.Scope, extra)
	}
	extraVars["security_group_name"] = serv.SecurityGroup
	if serv.TargetGroup == "" && serv.WebserverPort != nil {
		serv.TargetGroup = fmt.Sprintf("%s%s-tg", sys.Scope, extra)
	}
	extraVars["target_group_name"] = serv.TargetGroup
	serv.Vars = extraVars

	if serv.Vars["security_group_rules"] == nil {
		serv.Vars["security_group_rules"] = []ansible.SecurityGroupRule{}
	}
	if serv.WebserverPort != nil && *serv.WebserverPort > 0 {
		sgr := ansible.SecurityGroupRule{
			Proto:    "tcp",
			FromPort: strconv.Itoa(*serv.WebserverPort),
			ToPort:   strconv.Itoa(*serv.WebserverPort),
			Group:    sys.LoadbalancerGroup,
		}
		serv.Vars["security_group_rules"] = append(serv.Vars["security_group_rules"].([]ansible.SecurityGroupRule), sgr)
		serv.Vars["webserver_port"] = strconv.Itoa(*serv.WebserverPort)
	}
}

func NodeProvision(serv system.Service, nerthusHost, nodeDir, name string, nodeNum int) (play []byte, err error) {
	serv.Node.Vars["nerthus_host"] = nerthusHost
	if bootstrap && serv.ServiceInfo.Name == "Nerthus" {
		serv.Node.Vars["git_token"] = gitToken
		serv.Node.Vars["git_repo"] = gitRepo
		serv.Node.Vars["boot_env"] = bootstrapEnv
		out, err = GenerateNodePlay(envFS, configDir, serv, name, nodeNum)
		if err != nil {
			log.WithError(err).Fatal("while generating node play")
		}
		serv.Vars["bootstrap"] = `cat <<'EOF' > bootstrap.yml
{{ lookup('file', 'nodes/` + name + `_bootstrap.yml') }}
EOF
su -c "ansible-playbook bootstrap.yml" ec2-user`
		fn = fmt.Sprintf("%s/%s_bootstrap.yml", nodeDir, name)
		os.Remove(fn)
		os.WriteFile(fn, out, 0644)
	} else {
		out, err = GenerateNodePlay(envFS, configDir, serv, name, nodeNum)
		if err != nil {
			log.WithError(err).Fatal("while generating node play")
		}
		ha, ok := hostActions.Get(name)
		if !ok {
			ha = make(chan message.Action, 10)
			hostActions.Set(name, ha)
		}
		ha <- message.Action{
			Action:          message.Playbook,
			AnsiblePlaybook: out,
		}
	}
	return
}

func Test(serv system.Service, nerthusHost, gitToken, gitRepo, bootstrapEnv, ansibleDir, configDir string, envFS fs.FS, bootstrap bool) {
	nodeDir := filepath.Clean(fmt.Sprintf("%s/nodes", ansibleDir))
	for i, name := range serv.NodeNames {
	}
}

func ExecutePrivisioning(envFS fs.FS, dir string, serv system.Service, bufPool *sync.Pool, configDir string, ctx context.Context) {
	if bootstrap && serv.ServiceInfo.Name != "Nerthus" {
		return
	}
	buff := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buff)

	exec := execute.NewDefaultExecute(
		execute.WithWrite(io.Writer(buff)),
	)

	play := "provision.yml"
	if serv.Playbook != "" {
		play = fmt.Sprintf("%s/%s", serv.Playbook, play)
	}
	//TODO: Generate ansible files

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		ExtraVars: serv.Vars,
	}
	pb := &playbook.AnsiblePlaybookCmd{
		Playbooks:      []string{dir + play},
		Exec:           exec,
		StdoutCallback: "json",
		Options:        ansiblePlaybookOptions,
	}

	err := pb.Run(ctx)
	if err != nil {
		log.WithError(err).Error("while running ansible playbook")
		//return
	}

	res, err := results.ParseJSONResultsStream(io.Reader(buff))
	if err != nil {
		log.WithError(err).Fatal("while parsing json result stream")
		//panic(err)
	}

	for _, play := range res.Plays {
		for _, task := range play.Tasks {
			for _, content := range task.Hosts {
				status := "Finished"
				if content.Changed {
					status = "Changed"
				} else if content.Failed {
					status = "Failed"
				} else if content.Skipped {
					status = "Skipped: " + content.SkipReason
				}
				log.Info(task.Task.Name, "status", status, "output", fmt.Sprint(content.Msg))
				if status == "Failed" {
					log.Fatal("runbook failed", "stdout", content.StdoutLines, "stderr", content.StderrLines)
				}
			}
		}
	}
}
*/
