package systems

import (
	"fmt"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/executors/ansible"
	"github.com/cantara/nerthus2/system"
	"github.com/cantara/nerthus2/system/service"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var builtinRoleCash = make(map[string]ansible.Role)
var loadBuiltin sync.Once

func BuiltinRoles(efs fs.FS) (roles map[string]ansible.Role, err error) {
	if len(builtinRoleCash) != 0 {
		return builtinRoleCash, nil
	}
	loadBuiltin.Do(func() {
		err = ansible.ReadRoleDir(efs, "roles", builtinRoleCash)
	})
	if err != nil {
		return
	}
	return builtinRoleCash, nil
}

func Environment(env string, builtinRoles map[string]ansible.Role) (config system.Environment, err error) {
	dir := filepath.Clean("systems/" + env)

	envFS := os.DirFS(dir)
	config, err = LoadConfig[system.Environment](dir)
	if err != nil {
		return
	}
	config.FS = envFS
	config.Dir = dir
	config.SystemConfigs = map[string]system.System{}
	config.Roles = map[string]ansible.Role{}
	for k, v := range builtinRoles {
		config.Roles[k] = v
	}
	err = ansible.ReadRoleDir(envFS, "roles", config.Roles)
	if err != nil {
		return
	}
	return
}

func System(env system.Environment, systemDir string) (config system.System, err error) {
	dir := filepath.Clean(fmt.Sprintf("%s/systems/%s", env.Dir, systemDir))

	sysFS := os.DirFS(dir)
	config, err = LoadConfig[system.System](dir)
	if err != nil {
		return
	}
	config.FS = sysFS
	config.Dir = dir
	if config.OSName == "" {
		config.OSName = env.OSName
	}
	if config.OSArch == "" {
		config.OSArch = env.OSArch
	}
	if config.InstanceType == "" {
		config.InstanceType = env.InstanceType
	}
	config.Roles = map[string]ansible.Role{}
	for k, v := range env.Roles {
		config.Roles[k] = v
	}
	err = ansible.ReadRoleDir(sysFS, "roles", config.Roles)
	if err != nil {
		return
	}
	if config.Name == "" {
		if len(config.Services) > 1 {
			log.Fatal("No system name and more than one service in the system")
		}
		config.Name = config.Services[0].Name
	}
	if config.Scope == "" {
		config.Scope = fmt.Sprintf("%s-%s", env.Name, config.Name)
	}
	if config.VPC == "" {
		config.VPC = fmt.Sprintf("%s-vpc", config.Scope)
	}
	if config.Key == "" {
		config.Key = fmt.Sprintf("%s-key", config.Scope)
	}
	if config.Loadbalancer == "" {
		config.Loadbalancer = fmt.Sprintf("%s-lb", config.Scope)
	}
	if config.LoadbalancerGroup == "" {
		config.LoadbalancerGroup = fmt.Sprintf("%s-sg", config.Loadbalancer)
	}
	return
}

func serviceBase(sys system.System, serv *system.Service) (err error) {
	if serv.Generated == true {
		return
	}
	var extra string
	if sys.Name != serv.Name {
		extra = fmt.Sprintf("-%s", serv.Name)
	}
	if serv.OSName == "" {
		serv.OSName = sys.OSName
	}
	if serv.OSArch == "" {
		serv.OSArch = sys.OSArch
	}
	if serv.InstanceType == "" {
		serv.InstanceType = sys.InstanceType
	}
	if serv.SecurityGroup == "" {
		serv.SecurityGroup = fmt.Sprintf("%s%s-sg", sys.Scope, extra)
	}
	if serv.TargetGroup == "" && serv.WebserverPort != nil {
		serv.TargetGroup = fmt.Sprintf("%s%s-tg", sys.Scope, extra)
	}
	if serv.ClusterName == "" {
		serv.ClusterName = fmt.Sprintf("%s.%s", serv.Name, sys.Zone)
	}
	serv.ClusterInfo = map[string]system.ClusterInfo{}
	serv.Roles = map[string]ansible.Role{}
	for k, v := range sys.Roles {
		serv.Roles[k] = v
	}
	if serv.Local != "" {
		serv.ServiceInfo, err = LocalService(sys.Dir, serv)
		if err != nil {
			return
		}
	} else {
		serv.ServiceInfo, err = GitService(serv)
		if err != nil {
			return
		}
	}
	if serv.ServiceInfo.Artifact.Id == "" {
		serv.ServiceInfo.Artifact.Id = serv.Name
	}
	if serv.Properties != nil && !arrayContains(serv.ServiceInfo.Dependencies, "local_override") {
		serv.ServiceInfo.Dependencies = append(serv.ServiceInfo.Dependencies, "local_override")
	}
	if serv.Files != nil && !arrayContains(serv.ServiceInfo.Dependencies, "service_files") {
		serv.ServiceInfo.Dependencies = append(serv.ServiceInfo.Dependencies, "service_files")
	}
	if serv.NumberOfNodes == 0 {
		if serv.ServiceInfo.Requirements.NotClusterAble {
			serv.NumberOfNodes = 1
		} else {
			serv.NumberOfNodes = 3
		}
	}
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
		err = ErrMissMatchNumberOfNamesAndProvidedNames
		log.WithError(err).Error("while creating service config", "numberOfNodes", serv.NumberOfNodes, "nodeNames", serv.NodeNames)
		return
	}

	if serv.WebserverPort != nil && *serv.WebserverPort > 0 {
		serv.SecurityGroupRules = []ansible.SecurityGroupRule{
			{
				Proto:    "tcp",
				FromPort: strconv.Itoa(*serv.WebserverPort),
				ToPort:   strconv.Itoa(*serv.WebserverPort),
				Group:    sys.LoadbalancerGroup,
			},
		}
	}
	serv.Generated = true
	return
}

func Service(sys system.System, serv *system.Service) (err error) {
	if !serv.Generated {
		err = serviceBase(sys, serv)
		if err != nil {
			return
		}
	}

	for _, fromServ := range sys.Services {
		for from, to := range fromServ.Override {
			if serv.Name != to {
				continue
			}
			if len(serv.Expose) == 0 {
				err = ErrOverrideDoesNotExportAnyPorts
				log.WithError(err).Error("while setting override security group rules", "from", from, "to", to)
				return
			}
			if !fromServ.Generated {
				err = serviceBase(sys, fromServ)
				if err != nil {
					return
				}
			}
			sgrs := make([]ansible.SecurityGroupRule, len(serv.Expose))
			i := 0
			for _, v := range serv.Expose {
				sgrs[i] = ansible.SecurityGroupRule{
					Proto:    "tcp",
					FromPort: strconv.Itoa(v),
					ToPort:   strconv.Itoa(v),
					Group:    fromServ.SecurityGroup,
				}
				i++
			}
			serv.SecurityGroupRules = append(serv.SecurityGroupRules, sgrs...)
			fromServ.ClusterInfo[serv.Name] = system.ClusterInfo{
				Name:  serv.ClusterName,
				Ports: serv.Expose,
			}
		}
	}
	return
}

var ErrMissMatchNumberOfNamesAndProvidedNames = errors.New("provided node names does not match number of nodes")
var ErrOverrideDoesNotExportAnyPorts = errors.New("trying to connect to a service that does not expose any ports")

func GitService(serv *system.Service) (servInfo *service.Service, err error) {
	if serv.Git == "" {
		return
	}
	if serv.Branch == "" {
		serv.Branch = "main"
	}
	u, err := url.Parse(fmt.Sprintf("https://%s/%s/nerthus.yml", strings.ReplaceAll(serv.Git, "github", "raw.githubusercontent"), serv.Branch))
	if err != nil {
		err = errors.Wrap(err, "creating url for service info")
		return
	}
	log.Trace("GetService", "url", u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("getting service info from git failed, status: %d", resp.StatusCode)
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
	servInfo = &service.Service{}
	err = yaml.Unmarshal(data, servInfo)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("getting service info from git, %s:%s", "url", u.String()))
		return
	}
	return
}

func LocalService(systemDir string, serv *system.Service) (servInfo *service.Service, err error) {
	if serv.Local == "" {
		return
	}
	data, err := os.ReadFile(filepath.Clean(fmt.Sprintf("%s/services/%s", systemDir, serv.Local)))
	if err != nil {
		err = errors.Wrap(err, "unable to read local service file")
		return
	}
	servInfo = &service.Service{}
	err = yaml.Unmarshal(data, servInfo)
	if err != nil {
		err = errors.Wrap(err, "unable to unmarshal local service file")
		return
	}
	return
}

func arrayContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
