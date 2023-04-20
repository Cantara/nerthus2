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
	if config.Nerthus == "" {
		config.Nerthus = fmt.Sprintf("nerthus.%s", config.Domain)
	}
	if config.Visuale == "" {
		config.Visuale = fmt.Sprintf("visuale.%s", config.Domain)
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
		if len(config.Clusters) > 1 {
			log.Fatal("No system name and more than one service in the system")
		}
		config.Name = config.Clusters[0].Name
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
	if config.RoutingMethod == "" {
		config.RoutingMethod = system.RoutingPath
	}
	if config.Loadbalancer == "" {
		config.Loadbalancer = fmt.Sprintf("%s-lb", config.Scope)
	}
	if config.LoadbalancerGroup == "" {
		config.LoadbalancerGroup = fmt.Sprintf("%s-sg", config.Loadbalancer)
	}
	if config.Zone == "" {
		config.Zone = strings.ToLower(fmt.Sprintf("%s.%s.infra", config.Name, env.Name))
	}
	if config.Domain == "" {
		config.Domain = env.Domain
	}
	return
}

func clusterBase(sys system.System, clust *system.Cluster) (err error) {
	if clust.Generated == true {
		return
	}
	var extra string
	if sys.Name != clust.Name {
		extra = fmt.Sprintf("-%s", clust.Name)
	}
	if clust.OSName == "" {
		clust.OSName = sys.OSName
	}
	if clust.OSArch == "" {
		clust.OSArch = sys.OSArch
	}
	if clust.InstanceType == "" {
		clust.InstanceType = sys.InstanceType
	}
	if clust.SecurityGroup == "" {
		clust.SecurityGroup = fmt.Sprintf("%s%s-sg", sys.Scope, extra)
	}
	if clust.TargetGroup == "" && clust.HasWebserverPort() {
		clust.TargetGroup = fmt.Sprintf("%s%s-tg", sys.Scope, extra)
	}
	if clust.ClusterName == "" {
		clust.ClusterName = fmt.Sprintf("%s.%s", clust.Name, sys.Zone)
	}
	clust.ClusterInfo = map[string]system.ClusterInfo{}
	clust.Roles = map[string]ansible.Role{}
	for k, v := range sys.Roles {
		clust.Roles[k] = v
	}
	if clust.NumberOfNodes == 0 {
		if clust.IsClusterAble() {
			clust.NumberOfNodes = 3
		} else {
			clust.NumberOfNodes = 1
		}
	}
	if len(clust.NodeNames) == 0 {
		if clust.NumberOfNodes == 1 {
			clust.NodeNames = []string{
				fmt.Sprintf("%s%s", sys.Scope, extra),
			}
		} else {
			clust.NodeNames = make([]string, clust.NumberOfNodes)
			for num := 1; num <= clust.NumberOfNodes; num++ {
				clust.NodeNames[num-1] = fmt.Sprintf("%s%s-%d", sys.Scope, extra, num)
			}
		}
	}
	if len(clust.NodeNames) != clust.NumberOfNodes {
		err = ErrMissMatchNumberOfNamesAndProvidedNames
		log.WithError(err).Error("while creating service config", "numberOfNodes", clust.NumberOfNodes, "nodeNames", clust.NodeNames)
		return
	}

	for _, serv := range clust.Services {
		if serv.WebserverPort != nil && *serv.WebserverPort > 0 {
			clust.SecurityGroupRules = []ansible.SecurityGroupRule{
				{
					Proto:    "tcp",
					FromPort: strconv.Itoa(*serv.WebserverPort),
					ToPort:   strconv.Itoa(*serv.WebserverPort),
					Group:    sys.LoadbalancerGroup,
				},
			}
		}
	}
	clust.Generated = true
	return
}

func Service(sys system.System, serv *system.Service) (err error) {
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
	if serv.Properties != nil && !arrayContains(serv.ServiceInfo.Requirements.Roles, "local_override") {
		serv.ServiceInfo.Requirements.Roles = append(serv.ServiceInfo.Requirements.Roles, "local_override")
	}
	if serv.Files != nil && !arrayContains(serv.ServiceInfo.Requirements.Roles, "service_files") {
		serv.ServiceInfo.Requirements.Roles = append(serv.ServiceInfo.Requirements.Roles, "service_files")
	}
	return
}

func Cluster(sys system.System, clust *system.Cluster) (err error) {
	if !clust.Generated {
		err = clusterBase(sys, clust)
		if err != nil {
			return
		}
	}

	for _, fromServ := range sys.Clusters {
		for from, to := range fromServ.Override {
			if clust.Name != to {
				continue
			}
			if len(clust.Expose) == 0 {
				err = ErrOverrideDoesNotExportAnyPorts
				log.WithError(err).Error("while setting override security group rules", "from", from, "to", to)
				return
			}
			if !fromServ.Generated {
				err = clusterBase(sys, fromServ)
				if err != nil {
					return
				}
			}
			sgrs := make([]ansible.SecurityGroupRule, len(clust.Expose))
			i := 0
			for _, v := range clust.Expose {
				sgrs[i] = ansible.SecurityGroupRule{
					Proto:    "tcp",
					FromPort: strconv.Itoa(v),
					ToPort:   strconv.Itoa(v),
					Group:    fromServ.SecurityGroup,
				}
				i++
			}
			clust.SecurityGroupRules = append(clust.SecurityGroupRules, sgrs...)
			fromServ.ClusterInfo[clust.Name] = system.ClusterInfo{
				Name:  clust.ClusterName,
				Ports: clust.Expose,
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
