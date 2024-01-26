package config

import (
	"errors"
	"fmt"
	"io/fs"
	iofs "io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config/schema"
)

func FindFilesAndSystems(dir string) (files, systems []string, err error) {
	err = filepath.WalkDir(dir, func(path string, d iofs.DirEntry, err error) error {
		if path == dir {
			return nil
		}
		path = strings.TrimPrefix(path, dir)[1:]
		base := filepath.Base(path)
		lowerBase := strings.ToLower(base)
		sbragi.Info("walking", "path", path, "base", lowerBase)
		if d.IsDir() {
			if strings.HasPrefix(base, ".") {

				return fs.SkipDir
			}
			switch lowerBase {
			case "packages":
				fallthrough
			case "packageManagers":
				fallthrough
			case "features":
				fallthrough
			case "services":
				err = filepath.WalkDir(filepath.Join(dir, path), func(path string, d iofs.DirEntry, err error) error {
					if d.IsDir() {
						return nil
					}
					if filepath.Ext(path) != ".cue" {
						return nil
					}
					path = strings.TrimPrefix(path, dir)[1:]
					sbragi.Info("walking", "path", path, "base", lowerBase, "ext", filepath.Ext(path))
					files = append(files, path)
					return nil
				})
			case "files":
			default:
				systems = append(systems, path)
			}
			return filepath.SkipDir
		}
		if filepath.Ext(path) != ".cue" {
			return nil
		}
		sbragi.Info("walking", "path", path, "base", lowerBase, "ext", filepath.Ext(path))
		files = append(files, path)
		return nil
	})
	return
}

func ParseSystem(files []string, system, root string) (conf Environment, err error) {
	//Might not need the copy as the pointers should be different outside and inside the function
	//systemFiles := make([]string, len(files))
	//copy(systemFiles, files)
	err = filepath.WalkDir(filepath.Join(root, system), func(path string, d iofs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".cue" {
			return nil
		}
		sbragi.Info("walking", "path", path, "base", strings.ToLower(filepath.Base(path)), "ext", filepath.Ext(path))
		path = strings.TrimPrefix(path, root)[1:]
		files = append(files, path)
		return nil
	})
	if err != nil {
		sbragi.WithError(err).Fatal("walked system dir", "root", root)
	}
	//TODO: get all subdirs with the exception of the files dir
	var tmp schema.Root
	sbragi.Info("loading", "root", root, "files", files)
	if err = schema.Load(root, files, &tmp); err != nil {
		sbragi.Info("hitt")
		return
	}
	sys := System{
		Name:          tmp.System.Name,
		MachineName:   tmp.System.MachineName,
		Cidr:          tmp.System.Cidr,
		Zone:          tmp.System.Zone,
		Domain:        tmp.System.Domain,
		RoutingMethod: tmp.System.RoutingMethod,
		Clusters:      make([]Cluster, len(tmp.System.Clusters)),
	}
	/*
		if root.System.Name != system {
			//Only parse the system we have actually read the files for
			err = fmt.Errorf("system name does not match directory name, %s!=%s", root.System.Name, system)
			return
		}
	*/
	for i, c := range tmp.System.Clusters {
		pm := map[string]schema.Package{}
		for k, v := range tmp.Packages {
			pm[k] = v
		}
		os, ok := tmp.OS[c.Node.Os]
		if !ok {
			err = fmt.Errorf("os is not present name=%s", c.Node.Os)
			return
		}
		sbragi.Info("cluste", "os", os, "oses", tmp.OS)
		pms := make([]PackageManager, len(os.PackageManagers))
		for i, name := range os.PackageManagers {
			pm, ok := tmp.PackageManagers[name]
			if !ok {
				err = fmt.Errorf("os is not present name=%s", c.Node.Os)
				return
			}
			pms[i] = PackageManager{
				Name:   name,
				Syntax: pm.Syntax,
				Local:  pm.Local,
				Root:   pm.Root,
			}
		}
		sys.Clusters[i] = Cluster{
			Name:        c.Name,
			MachineName: c.MachineName,
			Packages:    pm,
			Size:        c.Size,
			Internal:    c.Internal,
			Node: Node{
				Os: OS{
					Name:            c.Node.Os,
					PackageManagers: pms,
					Provides:        os.Provides,
				},
				Arch: c.Node.Arch,
				Size: c.Node.Size,
			},
			Services: make([]Service, len(c.Services)),
		}
		sbragi.Info("cluste", "node", sys.Clusters[i].Node)
		for j, serv := range c.Services {
		servReq:
			for _, name := range serv.Definition.Requirements.Services {
				nameLow := strings.ToLower(name)
				for _, clust := range tmp.System.Clusters {
					for _, serv := range clust.Services {
						if nameLow == strings.ToLower(serv.Name) {
							continue servReq
						}
					}
				}
				sbragi.Error("missing service requirement in system", "name", name)
			}
			var f []Feature
			f = featureToFeature(serv.Definition.Requirements.Features, tmp, &sys.Clusters[i], f)
			f = featureToFeature([]string{"tools", "utf8", "cron", "buri", "nerthus", "service " + serv.MachineName}, tmp, &sys.Clusters[i], f)
			if serv.Definition.Artifact.ID != "" {
				serv.Definition.APIPath = "/" + filepath.Clean(serv.Definition.APIPath)
				sbragi.Info("props", "path", serv.Definition.APIPath)
				tasks := []Task{
					{
						Info:    "Get and start service",
						Type:    "command",
						Command: []string{"buri", "run", "go", "-u", "-a", serv.Definition.Artifact.ID, "-g", serv.Definition.Artifact.Group},
					},
					{
						Info:    "Start service monitor",
						Type:    "command",
						Command: []string{"nerthus2-probe-health", "-d", "5m", "-r", fmt.Sprintf("https://%s/api/status/%s/%s/sf-nerthus?service_tag=SoftwareFactory&service_type=H2A", strings.TrimPrefix(tmp.VisualeURL, "https://"), sys.Name, serv.Name), "-h", fmt.Sprintf("http://localhost:%d%shealth", serv.Port, serv.Definition.APIPath), "-a", serv.Definition.Artifact.ID, "-t", serv.Definition.HealthType},
					},
				}
				if serv.Definition.Requirements.PropertiesName != "" {
					props, err := Props(&serv)
					if err != nil {
						sbragi.WithError(err).Error("while calculating properties")
						continue
					}
					sbragi.Info("props", "port", serv.Port, "path", serv.Definition.Requirements.PropertiesName)
					tasks = append([]Task{
						{
							Info: "Write propperties",
							Type: "file_string",
							Text: props,
							Dest: serv.Definition.Requirements.PropertiesName,
						},
					}, tasks...)
				}
				f = append(f, Feature{
					Name:     "Service",
					Friendly: "Settup Service",
					Tasks:    tasks,
				})
			} else {
			}
			p := make([]Package, len(serv.Definition.Requirements.Packages))
			for i, name := range serv.Definition.Requirements.Packages {
				def, ok := tmp.Packages[name]
				if !ok {
					err = fmt.Errorf("package is not present name=%s", name)
					return
				}
				p[i] = Package{
					Name:     name,
					Managers: def.Managers,
					//Provides: def.Provides,
				}
			}
			sys.Clusters[i].Services[j] = Service{
				Name:        serv.Name,
				MachineName: serv.MachineName,
				//MachineName: strings.ToLower(strings.ReplaceAll(serv.Name, " ", "_")),
				Props: serv.Props,
				Port:  serv.Port,
				Definition: ServiceInfo{
					Name:        serv.Definition.Name,
					MachineName: serv.Definition.MachineName,
					ServiceType: serv.Definition.ServiceType,
					HealthType:  serv.Definition.HealthType,
					APIPath:     serv.Definition.APIPath,
					Artifact:    Artifact(serv.Definition.Artifact),
					Requirements: Requirements{
						RAM:              serv.Definition.Requirements.RAM,
						Disk:             serv.Definition.Requirements.Disk,
						CPU:              serv.Definition.Requirements.CPU,
						PropertiesName:   serv.Definition.Requirements.PropertiesName,
						WebserverPortKey: serv.Definition.Requirements.WebserverPortKey,
						NotClusterAble:   serv.Definition.Requirements.NotClusterAble,
						IsFrontend:       serv.Definition.Requirements.IsFrontend,
						Features:         f,
						Packages:         p,
						Services:         serv.Definition.Requirements.Services,
					},
				},
			}
		}
	}
	conf = Environment{
		Name:        tmp.Name,
		MachineName: tmp.MachineName,
		NerthusURL:  tmp.NerthusURL,
		VisualeURL:  tmp.VisualeURL,
		System:      sys,
	}
	return
}

func contains[T comparable](arr []T, v T) int {
	for i, el := range arr {
		if el != v {
			continue
		}
		return i
	}
	return -1
}

func containsCompare[T any](arr []T, eq func(v T) bool) int {
	for i, el := range arr {
		if !eq(el) {
			continue
		}
		return i
	}
	return -1
}

func confTaskToTask(task schema.Task, cfg schema.Root, clust Cluster) Task {
	RequiresRoot := task.Privelage == "root"
	switch task.Type {
	case "install":
		def, ok := clust.Packages[task.Package]
		if !ok {
			sbragi.Error("package is not present", "name", task.Package)
			//continue system
			return Task{}
		}
		sbragi.Info("install", "managers", def.Managers)
		return Task{
			Info:    task.Info,
			Type:    task.Type,
			Root:    RequiresRoot,
			Manager: task.Manager,
			Package: &Package{
				Name:     task.Package,
				Managers: def.Managers,
				//Provides: def.Provides,
			},
		}
	case "install_local":
		return Task{
			Info:    task.Info,
			Type:    task.Type,
			Root:    RequiresRoot,
			Manager: task.Manager,
			File:    task.File,
		}
	case "install_external":
		return Task{
			Info:    task.Info,
			Type:    task.Type,
			Root:    RequiresRoot,
			Manager: task.Manager,
			Url:     task.Url,
		}
	case "download":
		return Task{
			Info:   task.Info,
			Type:   task.Type,
			Root:   RequiresRoot,
			Source: task.Source,
			Dest:   task.Dest,
		}
	case "link":
		return Task{
			Info:   task.Info,
			Type:   task.Type,
			Root:   RequiresRoot,
			Source: task.Source,
			Dest:   task.Dest,
		}
	case "delete":
		return Task{
			Info: task.Info,
			Type: task.Type,
			File: task.File,
		}
	case "enable":
		return Task{
			Info:    task.Info,
			Type:    task.Type,
			Root:    RequiresRoot,
			Service: task.Service,
			Start:   task.Start,
		}
	case "schedule":
		return Task{
			Info: task.Info,
			Type: task.Type,
			Root: RequiresRoot,
		}
	case "file_string":
		return Task{
			Info: task.Info,
			Type: task.Type,
			Root: RequiresRoot,
			Text: task.Text,
			Dest: task.Dest,
		}
	case "file_bytes":
		return Task{
			Info: task.Info,
			Type: task.Type,
			Root: RequiresRoot,
			Data: task.Data,
			Dest: task.Dest,
		}
	case "user":
		return Task{
			Info:     task.Info,
			Type:     task.Type,
			Root:     RequiresRoot,
			Username: task.Username,
		}
	case "command":
		return Task{
			Info:    task.Info,
			Type:    task.Type,
			Root:    RequiresRoot,
			Command: task.Command,
		}
	default:
		sbragi.Warning("unsuported task", "type", task.Type)
	}
	return Task{}
}

func featureToFeature(features []string, cfg schema.Root, cluster *Cluster, serviceFeats []Feature) []Feature {
	for _, name := range features {
		var service string
		if strings.HasPrefix(name, "service ") {
			s := strings.SplitN(name, " ", 2)
			name, service = s[0], s[1]
		}
		feat, ok := cfg.Features[name]
		if !ok {
			sbragi.Error("feature is not present", "name", name)
			return nil
		}
		if service != "" {
			name = fmt.Sprintf("%s %s", name, service)
			r := strings.NewReplacer("<service>", service)
			tmp := feat
			feat = schema.Feature{
				Friendly: r.Replace(feat.Friendly),
				Requires: feat.Requires,
				Packages: feat.Packages,
				Tasks:    make([]schema.Task, len(feat.Tasks)),
				Custom:   make(map[string][]schema.Task),
			}
			for i := range tmp.Tasks {
				task := tmp.Tasks[i]
				switch task.Type {
				case "download":
					task.Source = r.Replace(task.Source)
					task.Dest = r.Replace(task.Dest)
				case "link":
					task.Source = r.Replace(task.Source)
					task.Dest = r.Replace(task.Dest)
				case "delete":
					task.File = r.Replace(task.File)
				case "file_string":
					task.Text = r.Replace(task.Text)
					task.Dest = r.Replace(task.Dest)
				case "file_bytes":
					task.Dest = r.Replace(task.Dest)
				case "user":
					task.Username = r.Replace(task.Username)
				case "command":
					for i := range task.Command {
						task.Command[i] = r.Replace(task.Command[i])
					}
				}
				feat.Tasks[i] = task
			}
			for k, v := range tmp.Custom {
				tasks := make([]schema.Task, len(v))
				for i := range v {
					task := v[i]
					switch task.Type {
					case "download":
						task.Source = r.Replace(task.Source)
						task.Dest = r.Replace(task.Dest)
					case "link":
						task.Source = r.Replace(task.Source)
						task.Dest = r.Replace(task.Dest)
					case "delete":
						task.File = r.Replace(task.File)
					case "file_string":
						task.Text = r.Replace(task.Text)
						task.Dest = r.Replace(task.Dest)
					case "file_bytes":
						task.Dest = r.Replace(task.Dest)
					case "user":
						task.Username = r.Replace(task.Username)
					case "command":
						for i := range task.Command {
							task.Command[i] = r.Replace(task.Command[i])
						}
					}
					tasks[i] = task
				}
				feat.Custom[k] = tasks
			}
		}
		var feats *[]Feature
		if feat.Service(cluster.Node.Os.Name) {
			feats = &serviceFeats
		} else {
			feats = &cluster.System
		}
		if containsCompare(*feats, func(v Feature) bool { return v.Name == name }) >= 0 {
			continue
		}
		org := len(*feats)
		serviceFeats = featureToFeature(feat.Requires, cfg, cluster, serviceFeats)
		if feat.Service(cluster.Node.Os.Name) {
			feats = &serviceFeats
		} else {
			feats = &cluster.System
		}
		if containsCompare((*feats)[org:], func(v Feature) bool { return v.Name == name }) >= 0 {
			continue
		}
		var tasks []Task
		if cust, ok := feat.Custom[cluster.Node.Os.Name]; ok {
			tasks = make([]Task, len(cust))
			for tn, task := range cust {
				sbragi.Info("manager check", "install?", strings.HasPrefix(task.Type, "install"), "has?", containsCompare(cluster.Node.Os.PackageManagers, func(v PackageManager) bool { return v.Name == task.Manager }), "manager", task.Manager)
				switch task.Type {
				case "install":
					p, ok := cluster.Packages[task.Package]
					if !ok {
						sbragi.Error("required package does not exist", "package", task.Package)
						continue
					}
					serviceFeats = validateOrAddPackageManager(p.Managers, cluster, cfg, serviceFeats)
				case "installLocal":
					serviceFeats = validateOrAddPackageManager([]string{task.Manager}, cluster, cfg, serviceFeats)
				}
				tasks[tn] = confTaskToTask(task, cfg, *cluster)
			}
		} else {
			tasks = make([]Task, len(feat.Tasks))
			for tn, task := range feat.Tasks {
				sbragi.Info("manager check", "install?", strings.HasPrefix(task.Type, "install"), "has?", containsCompare(cluster.Node.Os.PackageManagers, func(v PackageManager) bool { return v.Name == task.Manager }), "manager", task.Manager)
				switch task.Type {
				case "install":
					p, ok := cluster.Packages[task.Package]
					if !ok {
						sbragi.Error("required package does not exist", "package", task.Package)
						continue
					}
					serviceFeats = validateOrAddPackageManager(p.Managers, cluster, cfg, serviceFeats)
				case "installLocal":
					serviceFeats = validateOrAddPackageManager([]string{task.Manager}, cluster, cfg, serviceFeats)
				}
				tasks[tn] = confTaskToTask(task, cfg, *cluster)
			}
		}
		*feats = append(*feats, Feature{
			Name:     name,
			Friendly: feat.Friendly,
			Tasks:    tasks,
		})
		for k, v := range feat.Packages {
			cluster.Packages[k] = v
		}
	}
	return serviceFeats
}

func Props(serv *schema.Service) (string, error) {
	serv.Definition.Requirements.WebserverPortKey = strings.TrimSpace(serv.Definition.Requirements.WebserverPortKey)
	if serv.Definition.Requirements.WebserverPortKey == "" && serv.Port > 0 {
		return "", ErrHasWebserverPortAndNoKey
	}
	if serv.Definition.Requirements.WebserverPortKey == "" {
		return serv.Props, nil
	}
	if serv.Port > 0 && serv.Port <= 1024 {
		return "", fmt.Errorf("Provided port is too small, port=%d", serv.Port)
	}
	lines := strings.Split(serv.Props, "\n")
	found := false
	for l, line := range lines {
		sbragi.Info("props", "line", line, "prefix", strings.HasPrefix(line, serv.Definition.Requirements.WebserverPortKey))
		if !strings.HasPrefix(line, serv.Definition.Requirements.WebserverPortKey) {
			continue
		}
		found = true
		if serv.Port <= 0 {
			var err error
			serv.Port, err = strconv.Atoi(strings.TrimSpace(strings.Split(line, "=")[1]))
			if err != nil {
				return "", fmt.Errorf("Missing webserver port but has a webserver key, err: %w", err)
			}
			if serv.Port <= 1024 {
				return "", fmt.Errorf("Provided port is too small, port=%d", serv.Port)
			}
			sbragi.Info("props", "port", serv.Port)
			break //Not checking for multiple occurances
		}
		sbragi.Info("props", "port", serv.Port)
		lines[l] = fmt.Sprintf("%s=%d", serv.Definition.Requirements.WebserverPortKey, serv.Port)
		break
	}
	if found {
		return strings.Join(lines, "\n"), nil
	}
	sbragi.Info("props", "port", serv.Port)
	if serv.Port <= 1024 {
		return "", ErrHasWebserverKeyAndNoPort
	}
	return fmt.Sprintf("%s=%d\n%s", serv.Definition.Requirements.WebserverPortKey, serv.Port, serv.Props), nil
}

func validateOrAddPackageManager(managers []string, cluster *Cluster, cfg schema.Root, serviceFeats []Feature) []Feature {
	if containsCompare(cluster.Node.Os.PackageManagers, func(v PackageManager) bool {
		for _, manager := range managers {
			if v.Name == manager {
				return true
			}
		}
		return false
	}) < 0 {
		var ok bool
		for i := 0; i < len(managers) && !ok; i++ {
			var pm schema.PackageManager
			pm, ok = cfg.PackageManagers[managers[i]]
			if !ok {
				sbragi.Error("required package manager does not exist", "manger", managers[i])
				continue
			}
			if len(pm.Requires) == 0 {
				sbragi.Error("required package manager can not be installed, does not require any features")
				continue
			}
			serviceFeats = featureToFeature(pm.Requires, cfg, cluster, serviceFeats)
			cluster.Node.Os.PackageManagers = append(cluster.Node.Os.PackageManagers, PackageManager{
				Name:   managers[i],
				Syntax: pm.Syntax,
				Local:  pm.Local,
			})
		}
	}
	return serviceFeats
}

var ErrHasWebserverKeyAndNoPort = errors.New("webserver key provided without providing port")
var ErrHasWebserverPortAndNoKey = errors.New("webserver port provided without providing webserver_port_key")
