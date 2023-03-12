package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/ansible"
	"github.com/cantara/nerthus2/system"
	"github.com/cantara/nerthus2/system/service"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

func Execute(dir string) {
	var finishedWG sync.WaitGroup
	defer finishedWG.Wait()
	roles := make(map[string]ansible.Role)
	err := ReadRoleDir(EFS, "roles", roles)
	if err != nil {
		log.WithError(err).Fatal("while reading env roles")
	}
	envFS := os.DirFS(dir)
	env, err := LoadConfig[system.Environment](dir)
	if err != nil {
		log.WithError(err).Fatal("while loading environment config")
	}
	envRoles := make(map[string]ansible.Role)
	for k, v := range roles {
		envRoles[k] = v
	}
	err = ReadRoleDir(envFS, "roles", envRoles)
	if err != nil {
		log.WithError(err).Fatal("while reading env roles")
	}

	for _, systemDir := range env.Systems {
		sys := BuildSystemSetup(envFS, env, envRoles, systemDir, dir)

		finishedWG.Add(1)
		go func() {
			defer finishedWG.Done()
			rules, defaultAction := BuildLoadbalancerSetup(sys)
			var wg sync.WaitGroup
			for _, serv := range sys.Services {
				if serv.Playbook != "" {
					wg.Wait()
					ExecutePrivisioning(fmt.Sprintf("%s/%s/", dir, serv.Playbook), serv, &bufPool)
					continue
				}
				wg.Add(1)
				go func(serv system.Service) {
					ExecutePrivisioning(dir+"/ansible/", serv, &bufPool)
					wg.Done()
				}(serv)
			}
			wg.Wait()
			ExecuteLoadbalancer(dir, rules, defaultAction, sys, env)
		}()
	}
}

func LoadConfig[T any](dir string) (out T, err error) {
	data, err := os.ReadFile(dir + "/config.yml")
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return
	}
	return
}

func BuildSystemSetup(envFS fs.FS, env system.Environment, roles map[string]ansible.Role, systemDir, dir string) (sys system.System) {
	systemRoles := make(map[string]ansible.Role)
	for k, v := range roles {
		systemRoles[k] = v
	}
	err := ReadRoleDir(envFS, systemDir+"/roles", systemRoles)
	if err != nil {
		log.WithError(err).Fatal("while reading system roles")
	}
	sys, err = LoadConfig[system.System](fmt.Sprintf("%s/%s", dir, systemDir))
	if err != nil {
		log.WithError(err).Fatal("while loading system config")
	}

	nameBase := sys.Name
	if nameBase == "" {
		if len(sys.Services) > 1 {
			log.Fatal("No system name and more than one service in the system")
		}
		nameBase = sys.Services[0].Name
	}
	if sys.Scope == "" {
		sys.Scope = fmt.Sprintf("%s-%s", env.Name, nameBase)
	}
	if sys.VPC == "" {
		sys.VPC = fmt.Sprintf("%s-vpc", sys.Scope)
	}
	if sys.Key == "" {
		sys.Key = fmt.Sprintf("%s-key", sys.Scope)
	}
	if sys.Loadbalancer == "" {
		sys.Loadbalancer = fmt.Sprintf("%s-lb", sys.Scope)
	}
	if sys.LoadbalancerGroup == "" {
		sys.LoadbalancerGroup = fmt.Sprintf("%s-sg", sys.Loadbalancer)
	}
	for i, serv := range sys.Services {
		if serv.Git == "" && serv.Local == "" {
			continue
		}
		if serv.Git != "" && serv.Branch == "" {
			log.Fatal("missing branch when getting from git", "service", serv)
			continue //Only in case fatal gets changed to error
		}
		var serviceInfo service.Service
		if serv.Local != "" {
			bdata, err := os.ReadFile(fmt.Sprintf("%s/%s/%s", dir, systemDir, serv.Local))
			if err != nil {
				log.WithError(err).Fatal("unable to read local service file")
				continue
			}
			err = yaml.Unmarshal(bdata, &serviceInfo)
			if err != nil {
				log.WithError(err).Fatal("unable to unmarshal local service file")
				continue
			}
		} else {
			u, err := url.Parse(fmt.Sprintf("https://%s/%s/nerthus.yml", strings.ReplaceAll(serv.Git, "github", "raw.githubusercontent"), serv.Branch))
			if err != nil {
				log.WithError(err).Fatal("while creating url for service info")
				continue
			}
			serviceInfo, err = GetService(u)
			if err != nil {
				log.WithError(err).Fatal("while getting service info from git", "url", u.String())
				continue
			}
		}
		sys.Services[i].ServiceInfo = &serviceInfo
		if sys.Services[i].NumberOfNodes == 0 { //TODO: actually handle requirements
			if serviceInfo.Requirements.NotClusterAble {
				sys.Services[i].NumberOfNodes = 1
			} else {
				sys.Services[i].NumberOfNodes = 3
			}
		}
		sys.Services[i].Node = &ansible.Playbook{
			Name:       serviceInfo.Name,
			Hosts:      "localhost",
			Connection: "local",
			Vars: map[string]string{
				"env":          env.Name,
				"service":      serviceInfo.Name,
				"service_type": serviceInfo.ServiceType,
				"health_type":  serviceInfo.HealthType,
			},
		}
		overrides := make([]string, len(serv.Override))
		oi := 0
		for k := range serv.Override {
			overrides[oi] = k
			oi++
		}
		var done []string
		for _, dep := range serviceInfo.Dependencies {
			if ArrayContains(overrides, dep) {
				continue
			}
			AddTask(dep, sys.Services[i].Node, &done, systemRoles)
		}

	}
	nerthusVars := map[string]string{
		"region": "ap-northeast-1",
	}

	for i, serv := range sys.Services {
		extraVars := map[string]any{}
		AddVars(serv.Node.Vars, extraVars)
		AddVars(env.Vars, extraVars)
		AddVars(sys.Vars, extraVars)
		AddVars(serv.Vars, extraVars)
		AddVars(nerthusVars, extraVars)

		var extra string
		if nameBase != serv.Name {
			extra = fmt.Sprintf("-%s", serv.Name)
		}

		extraVars["system"] = sys.Name
		extraVars["service"] = serv.Name
		extraVars["name_base"] = sys.Scope
		extraVars["vpc_name"] = sys.VPC
		extraVars["key_name"] = sys.Key
		extraVars["loadbalancer_name"] = sys.Loadbalancer
		extraVars["loadbalancer_group"] = sys.LoadbalancerGroup
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
		if serv.TargetGroup == "" {
			serv.TargetGroup = fmt.Sprintf("%s%s-tg", sys.Scope, extra)
		}
		extraVars["target_group_name"] = serv.TargetGroup
		sys.Services[i].Vars = extraVars

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
		}

		for _, v := range serv.Override {
			if !strings.HasPrefix(v, "services") {
				continue
			}
			overrideService := strings.ReplaceAll(v, "services/", "")
			for oi, overs := range sys.Services {
				if overs.Name != overrideService {
					continue
				}
				if len(overs.Expose) == 0 {
					log.Fatal("trying to connect to a service that does not expose any ports", "from", serv, "to", overs)
				}
				sys.Services[i].Vars[fmt.Sprintf("%s_host", overs.Name)] = fmt.Sprintf("%s.%s", overs.Name, sys.Services[i].Vars["zone"])
				sgr := ansible.SecurityGroupRule{
					Proto:    "tcp",
					FromPort: strconv.Itoa(overs.Expose[0]),
					ToPort:   strconv.Itoa(overs.Expose[0]),
					Group:    serv.SecurityGroup,
				}
				if overs.Vars["security_group_rules"] == nil {
					sys.Services[oi].Vars["security_group_rules"] = []ansible.SecurityGroupRule{}
				}
				sys.Services[oi].Vars["security_group_rules"] = append(sys.Services[oi].Vars["security_group_rules"].([]ansible.SecurityGroupRule), sgr)
			}
		}
	}
	return
}

func AddVars[T comparable](inVars map[string]T, outVars map[string]any) {
	for k, v := range inVars {
		var zero T
		if v == zero { //Excluding all zero values might not be optimal for items like ints.
			continue
		}
		outVars[k] = v
	}
}

func BuildLoadbalancerSetup(sys system.System) (rules []Rule, defaultAction []Action) {
	i := 0
	rules = []Rule{}
	if len(sys.Services) == 1 && sys.Services[0].ServiceInfo.Requirements.IsFrontend {
		defaultAction = []Action{
			{
				TargetGroupName: sys.Services[0].Vars["target_group_name"].(string),
				Type:            "forward",
			},
		}
	} else {
		for _, serv := range sys.Services {
			if serv.Playbook != "" {
				continue
			}
			i++
			rules = append(rules, Rule{
				Conditions: []Condition{
					{
						Field: "path-pattern",
						Values: []string{
							fmt.Sprintf("/%s", serv.Vars["service"]),
							fmt.Sprintf("/%s/*", serv.Vars["service"]),
						},
					},
				},
				Actions: []Action{
					{
						TargetGroupName: serv.Vars["target_group_name"].(string),
						Type:            "forward",
					},
				},
				Priority: i,
			})
		}
	}
	return
}

func ExecutePrivisioning(dir string, serv system.Service, bufPool *sync.Pool) {
	buff := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buff)
	os.Mkdir(dir+"nodes", 0750)

	if serv.Node != nil {
		for k := range serv.Node.Vars {
			cur, ok := serv.Vars[k]
			if !ok || cur == nil {
				continue
			}
			log.Info("server node vars", "key", k, "val", cur)
			serv.Node.Vars[k] = fmt.Sprint(cur)
		}
		for i, name := range serv.Vars["node_names"].([]string) {
			serv.Node.Vars["host"] = name
			serv.Node.Vars["server_number"] = strconv.Itoa(i)
			out, err := yaml.Marshal([]ansible.Playbook{
				*serv.Node,
			})
			if err != nil {
				log.WithError(err).Fatal("unable to marshall json for node playbook", "node", serv.Node)
			}
			fn := fmt.Sprintf("%snodes/%s.yml", dir, serv.Node.Vars["host"])
			os.Remove(fn)
			os.WriteFile(fn, out, 0644)
		}
	}

	exec := execute.NewDefaultExecute(
		execute.WithWrite(io.Writer(buff)),
	)

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		ExtraVars: serv.Vars,
	}
	play := "provision.yml"
	if serv.Playbook != "" {
		play = fmt.Sprintf("%s/%s", serv.Playbook, play)
	}
	play = "bootstrap-provision.yml"
	pb := &playbook.AnsiblePlaybookCmd{
		Playbooks:      []string{dir + play},
		Exec:           exec,
		StdoutCallback: "json",
		Options:        ansiblePlaybookOptions,
	}

	err := pb.Run(context.Background())
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

func ExecuteLoadbalancer(dir string, rules []Rule, defaultAction []Action, sys system.System, env system.Environment) {
	buff := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buff)

	exec := execute.NewDefaultExecute(
		execute.WithWrite(io.Writer(buff)),
	)
	scope := sys.Name
	log.Info("scope", "sys.Name", scope)
	if scope == "" {
		scope = sys.Services[0].Name
		log.Info("scope", "service", scope)
	}
	nameBase := fmt.Sprintf("%s-%s", env.Name, scope)
	extraVars := map[string]interface{}{
		"rules":             rules,
		"vpc_name":          fmt.Sprintf("%s-vpc", nameBase),
		"certificate_arn":   "arn:aws:acm:ap-northeast-1:217183500018:certificate/31f4a295-84f3-46b2-b9a6-96100d474e46", //TODO: move to use a var path from system
		"loadbalancer_name": fmt.Sprintf("%s-lb", nameBase),
	}
	if defaultAction != nil {
		extraVars["default_actions"] = defaultAction
	}
	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		ExtraVars: extraVars,
	}
	play := "loadbalancer.yml"
	pb := &playbook.AnsiblePlaybookCmd{
		Playbooks:      []string{dir + "/ansible/" + play},
		Exec:           exec,
		StdoutCallback: "json",
		Options:        ansiblePlaybookOptions,
	}

	err := pb.Run(context.Background())
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
