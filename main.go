package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/syncmap"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/ansible"
	"github.com/cantara/nerthus2/message"
	"github.com/cantara/nerthus2/system"
	"github.com/cantara/nerthus2/system/service"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

//go:embed roles bootstrap
var EFS embed.FS

var bootstrap bool
var gitRepo string
var gitToken string
var systemName string

func init() {
	const ( //TODO: Add bootstrap git as a separate command from bootstrap.
		defaultBootstrap  = false
		bootstrapUsage    = "tells nerthus to bootstrap itself into aws"
		defaultGitRepo    = "github.com/cantara/nerthus2"
		gitRepoUsage      = "github repository for solution config"
		defaultGitToken   = ""
		gitTokenUsage     = "github repository granular access token"
		defaultSystemName = "nerthus2"
		systemNameUsage   = "defines the system that Nerthus should use to provision itself"
	)
	flag.BoolVar(&bootstrap, "bootstrap", defaultBootstrap, bootstrapUsage)
	flag.BoolVar(&bootstrap, "b", defaultBootstrap, bootstrapUsage+" (shorthand)")
	flag.StringVar(&gitRepo, "git-repo", defaultGitRepo, gitRepoUsage)
	flag.StringVar(&gitRepo, "r", defaultGitRepo, gitRepoUsage+" (shorthand)")
	flag.StringVar(&gitToken, "git-token", defaultGitToken, gitTokenUsage)
	flag.StringVar(&gitToken, "t", defaultGitToken, gitTokenUsage+" (shorthand)")
	flag.StringVar(&systemName, "system-name", defaultSystemName, systemNameUsage)
	flag.StringVar(&systemName, "n", defaultSystemName, systemNameUsage+" (shorthand)")
}
func main() {
	flag.Parse()
	bootstrap = true
	gitToken = "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"
	gitRepo = "github.com/SindreBrurberg/nerthus-test-config"

	dir := "greps" //"tmp-test-dir"

	bufPool := sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}
	fsdes, err := EFS.ReadDir("roles")
	if err != nil {
		panic(err)
	}
	roles := make(map[string]ansible.Role)
	for _, de := range fsdes {
		name := strings.TrimSuffix(de.Name(), ".yml")
		log.Info("roles/" + de.Name())
		log.Info(name)
		b, err := EFS.ReadFile("roles/" + de.Name())
		if err != nil {
			log.WithError(err).Fatal("while reading file in roles")
		}
		var role ansible.Role
		err = yaml.Unmarshal(b, &role)
		if err != nil {
			log.WithError(err).Fatal("while unmarshalling roles")
		}
		role.Id = name
		roles[name] = role
	}
	envFS := os.DirFS(dir)
	var env system.Environment
	data, err := os.ReadFile(dir + "/config.yml")
	if err != nil {
		log.WithError(err).Fatal("while reading environment config file")
	}
	err = yaml.Unmarshal(data, &env)
	if err != nil {
		log.WithError(err).Fatal("while unmarshalling environment config file")
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
		systemRoles := make(map[string]ansible.Role)
		for k, v := range envRoles {
			systemRoles[k] = v
		}
		err = ReadRoleDir(envFS, systemDir+"/roles", systemRoles)
		if err != nil {
			log.WithError(err).Fatal("while reading system roles")
		}
		var sys system.System
		data, err := os.ReadFile(fmt.Sprintf("%s/%s/config.yml", dir, systemDir))
		if err != nil {
			log.WithError(err).Fatal("while reading example config file")
		}
		err = yaml.Unmarshal(data, &sys)
		if err != nil {
			log.WithError(err).Fatal("while unmarshalling example config file")
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
			//"ami":    "ami-0bba69335379e17f8",
		}

		for i, serv := range sys.Services {
			extraVars := map[string]any{
				"system":  sys.Name,
				"service": serv.Name,
			}
			if serv.Node != nil {
				for k, v := range serv.Node.Vars {
					if k == "service" {
						continue
					}
					if k == "system" {
						continue
					}
					if v == "" {
						continue
					}
					extraVars[k] = v
				}
			}
			for k, v := range env.Vars {
				if v == "" {
					continue
				}
				extraVars[k] = v
			}
			for k, v := range sys.Vars {
				if v == "" {
					continue
				}
				extraVars[k] = v
			}
			for k, v := range serv.Vars {
				if v == "" {
					continue
				}
				extraVars[k] = v
			}
			for k, v := range nerthusVars {
				if v == "" {
					continue
				}
				extraVars[k] = v
			}

			scope := sys.Name
			log.Info("scope", "sys.Name", scope)
			var extra string
			if scope == "" {
				scope = sys.Services[0].Name
				log.Info("scope", "service", scope)
			} else {
				extra = fmt.Sprintf("-%s", extraVars["service"])
			}
			nameBase := fmt.Sprintf("%s-%s", env.Name, scope)
			extraVars["name_base"] = nameBase
			if v, ok := extraVars["key_name"]; !ok || v == nil || v == "" {
				extraVars["key_name"] = fmt.Sprintf("%s-key", nameBase)
			}
			if v, ok := extraVars["vpc_name"]; !ok || v == nil || v == "" {
				extraVars["vpc_name"] = fmt.Sprintf("%s-vpc", nameBase)
			}
			log.Info("vars", "key_name", extraVars["key_name"], "vpc_name", extraVars["vpc_name"])
			if v, ok := extraVars["node_names"]; !ok || v == nil {
				if serv.NumberOfNodes == 1 {
					extraVars["node_names"] = []string{
						fmt.Sprintf("%s%s", nameBase, extra),
					}
				} else {
					nodeNames := make([]string, serv.NumberOfNodes)
					for num := 1; num <= serv.NumberOfNodes; num++ {
						nodeNames[num-1] = fmt.Sprintf("%s%s-%d", nameBase, extra, num)
					}
					extraVars["node_names"] = nodeNames
				}
			}
			if v, ok := extraVars["security_group_name"]; !ok || v == nil || v == "" {
				extraVars["security_group_name"] = fmt.Sprintf("%s%s-sg", nameBase, extra)
			}
			if v, ok := extraVars["target_group_name"]; !ok || v == nil || v == "" {
				extraVars["target_group_name"] = fmt.Sprintf("%s%s-tg", nameBase, extra)
			}
			if v, ok := extraVars["loadbalancer_name"]; !ok || v == nil || v == "" {
				extraVars["loadbalancer_name"] = fmt.Sprintf("%s-lb", nameBase)
			}
			sys.Services[i].Vars = extraVars

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
						Group:    fmt.Sprintf("%s-%s-sg", extraVars["system"], extraVars["service"]),
					}
					if overs.Vars["security_group_rules"] == nil {
						sys.Services[oi].Vars["security_group_rules"] = []ansible.SecurityGroupRule{sgr}
						continue
					}
					sys.Services[oi].Vars["security_group_rules"] = append(sys.Services[oi].Vars["security_group_rules"].([]ansible.SecurityGroupRule), sgr)
				}
			}
		}

		/*
		   Rules:
		     - Conditions:
		         - Field: path-pattern
		           Values:
		             - "/{{ item.service }}"
		             - "/{{ item.service }}/*"
		       Actions:
		         - TargetGroupName: "{{ item.target_group_name }}"
		           Type: forward
		       Priority: "{{ item.path_priority }}"
		*/
		i := 0
		rules := []Rule{}
		var defaultAction []Action
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

		var wg sync.WaitGroup
		for _, serv := range sys.Services {
			if serv.Playbook != "" {
				wg.Wait()
				AnsibleService(fmt.Sprintf("%s/%s/", dir, serv.Playbook), serv, &bufPool)
				continue
			}
			wg.Add(1)
			go func(serv system.Service) {
				AnsibleService(dir+"/ansible/", serv, &bufPool)
				wg.Done()
			}(serv)
		}
		wg.Wait()

		func() {
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

			err = pb.Run(context.Background())
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
		}()
	}

	log.Fatal("f")
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port))
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}
	serv.API.PUT("/provision/:artifactId", func(c *gin.Context) {
		artifactId := c.Param("artifactId")
		auth := webserver.GetAuthHeader(c)
		if auth == "" {
			webserver.ErrorResponse(c, "authorization not provided", http.StatusForbidden)
			return
		}
		if auth != os.Getenv("authkey") {
			webserver.ErrorResponse(c, "unauthorized", http.StatusUnauthorized)
			return
		}

		/*
			_, err := webserver.UnmarshalBody[service](c)
			if err != nil {
				webserver.ErrorResponse(c, err.Error(), http.StatusBadRequest)
				return
			}
		*/

		buff := bufPool.Get().(*bytes.Buffer)
		defer bufPool.Put(buff)

		exec := execute.NewDefaultExecute(
			execute.WithWrite(io.Writer(buff)),
		)

		ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
			ExtraVars: map[string]interface{}{
				"region":    "eu-west-3",             //"ap-northeast-1",
				"ami":       "ami-00575c0cbc20caf50", //"ami-0bba69335379e17f8",
				"cidr_base": "10.110.0",
				"service":   artifactId,
				"env":       "exoreaction-lab",
				"zone":      "lab.exoreaction.infra",
			},
		}

		pb := &playbook.AnsiblePlaybookCmd{
			Playbooks:      []string{"playbook.yml"},
			Exec:           exec,
			StdoutCallback: "json",
			Options:        ansiblePlaybookOptions,
		}

		err = pb.Run(context.TODO())
		if err != nil {
			log.WithError(err).Error("while running ansible playbook")
			webserver.ErrorResponse(c, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := results.ParseJSONResultsStream(io.Reader(buff))
		if err != nil {
			log.WithError(err).Fatal("while parsing json result stream")
		}

		msgOutput := struct {
			Host    string `json:"host"`
			Message string `json:"message"`
		}{}

		for _, play := range res.Plays {
			for _, task := range play.Tasks {
				for _, content := range task.Hosts {
					err = json.Unmarshal([]byte(fmt.Sprint(content.Stdout)), &msgOutput)
					if err != nil {
						panic(err)
					}

					fmt.Printf("[%s] %s\n", msgOutput.Host, msgOutput.Message)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "service added"})
		return
	})

	serv.API.PUT("/deploy/:artifactId", func(c *gin.Context) {
	})

	hc := make(chan message.Action, 10)
	hostActions.Set("testHost", hc)
	hc <- message.Action{
		Action: message.Playbook,
		AnsiblePlaybook: `---
- name: Test playbook
  hosts: localhost
  connection: local
  tasks:
    - name: Ansible | Print test
      debug:
        msg: "test print"
    - name: Ansible | Skipp me
      debug:
        msg: "test print"
      when: false
`,
		ExtraVars: nil,
	}

	websocket.Serve[message.Action](serv.API, "/probe/:host", nil, func(reader <-chan message.Action, writer chan<- websocket.Write[message.Action], p gin.Params, ctx context.Context) {
		host := p.ByName("host")
		log.Info("opening websocket", "host", host)
		defer log.Info("closed websocket", "host", host)
		go func() {
			hostChan, ok := hostActions.Get(host)
			if !ok {
				hostChan = make(chan message.Action, 10)
				hostActions.Set(host, hostChan)
			}
			for a := range hostChan {
				errChan := make(chan error, 1)
				action := websocket.Write[message.Action]{
					Data: a,
					Err:  errChan,
				}
				select {
				case <-ctx.Done():
					return
				case writer <- action:
					err := <-errChan
					if err != nil {
						log.WithError(err).Error("unable to write action to nerthus probe",
							"action_type", reflect.TypeOf(action))
						continue
					}
				}
			}
		}()

		for msg := range reader {
			if msg.Response == nil {
				log.Warning("read action response without response", "action", msg)
				continue
			}
			log.Info("response from action", "message", msg.Response.Message, "status", msg.Response.Status)
		}
	})

	serv.Run()
}

var hostActions = syncmap.New[chan message.Action]()

func ArrayContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func AddTask(role string, pb *ansible.Playbook, done *[]string, roles map[string]ansible.Role) {
	if ArrayContains(*done, role) {
		return
	}
	r, ok := roles[role]
	if !ok {
		return
	}
	for _, req := range r.Dependencies {
		AddTask(req.Role, pb, done, roles)
	}
	for vn, vv := range r.Vars {
		if pb.Vars[vn] != "" {
			continue
		}
		pb.Vars[vn] = vv
	}
	pb.Tasks = append(pb.Tasks, r.Tasks...)
	*done = append(*done, r.Id)
}

func GetService(u *url.URL) (serv service.Service, err error) {
	log.Trace("GetService", "url", u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("get miss")
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.WithError(err).Debug("while closing response body")
		}
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &serv)
	return
}

/*
	pb := []ansible.Playbook{
		{
			Name:       "Service test Playbook",
			Hosts:      "localhost",
			Connection: "local",
			Vars:       map[string]string{},
		},
	}
	done := make([]string, len(roles))
	i := 0
	for _, role := range roles {
		if ArrayContains(done, role.Id) {
			continue
		}
		for _, req := range role.Dependencies {
			if ArrayContains(done, req.Role) {
				continue
			}
			dep := roles[req.Role]
			for vn, vv := range dep.Vars {
				pb[0].Vars[vn] = vv
			}
			pb[0].Tasks = append(pb[0].Tasks, dep.Tasks...)
			done[i] = dep.Id
			i++
		}
		for vn, vv := range role.Vars {
			pb[0].Vars[vn] = vv
		}
		pb[0].Tasks = append(pb[0].Tasks, role.Tasks...)
		done[i] = role.Id
		i++
	}

	out, err := yaml.Marshal(pb)
	if err != nil {
		panic(err)
	}
	os.Remove("ansible/out.yml")
	os.WriteFile("ansible/out.yml", out, 0644)
	for _, task := range pb[0].Tasks {
		fmt.Println(task["name"])
	}
*/

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

func AnsibleService(dir string, serv system.Service, bufPool *sync.Pool) {
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

func ReadRoleDir(dir fs.FS, path string, roles map[string]ansible.Role) error {
	_, err := dir.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return fs.WalkDir(dir, path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(dir, path)
		if err != nil {
			return err
		}
		var role ansible.Role
		err = yaml.Unmarshal(b, &role)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(d.Name(), ".yml")
		role.Id = name
		roles[name] = role
		return nil
	})
}
