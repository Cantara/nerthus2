package actions

import (
	"fmt"
	"strings"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
	"github.com/cantara/nerthus2/probe/statemachine"
	jsoniter "github.com/json-iterator/go"
)

var DryRun = true
var IsRoot = false

func Next(cfg pconf.Environment) (state statemachine.State[pconf.Environment], err error) {
	if len(cfg.System.Cluster.System) == 0 {
		if len(cfg.System.Cluster.Services) == 0 {
			fmt.Println(jsoniter.ConfigDefault.MarshalToString(cfg))
			return
		} else if len(cfg.System.Cluster.Services[0].Definition.Requirements.Features) == 0 {
			cfg.System.Cluster.Services = cfg.System.Cluster.Services[1:]
			return Next(cfg)
		} else if len(cfg.System.Cluster.Services[0].Definition.Requirements.Features[0].Tasks) == 0 {
			cfg.System.Cluster.Services[0].Definition.Requirements.Features = cfg.System.Cluster.Services[0].Definition.Requirements.Features[1:]
			return Next(cfg)
		}
		//TODO: Might want to do separate logick for incrementing to next system if all tasks are done
		state = statemachine.State[pconf.Environment]{
			State: cfg.System.Cluster.Services[0].Definition.Requirements.Features[0].Tasks[0].Type,
			Data:  cfg,
		}
		return
	} else if len(cfg.System.Cluster.System[0].Tasks) == 0 {
		cfg.System.Cluster.System = cfg.System.Cluster.System[1:]
		return Next(cfg)
	}
	state = statemachine.State[pconf.Environment]{
		State: cfg.System.Cluster.System[0].Tasks[0].Type,
		Data:  cfg,
	}
	return
}

func Execute(t string, f func(cfg pconf.Environment, task config.Task, service string) error) (string, statemachine.Fn[pconf.Environment]) {
	return t, func(cfg pconf.Environment) (state statemachine.State[pconf.Environment], err error) {
		var feat config.Feature
		var service string
		if len(cfg.System.Cluster.System) > 0 {
			feat = cfg.System.Cluster.System[0]
		} else if len(cfg.System.Cluster.Services) > 0 &&
			len(cfg.System.Cluster.Services[0].Definition.Requirements.Features) > 0 {
			feat = cfg.System.Cluster.Services[0].Definition.Requirements.Features[0]
			service = cfg.System.Cluster.Services[0].Name
		}
		if len(feat.Tasks) == 0 {
			err = fmt.Errorf("feature has no tasks left when trying to %s", t)
			return
		}
		task := feat.Tasks[0]
		if task.Type != t {
			err = fmt.Errorf("task type != %[1]s when trying to %[1]s", t)
			return
		}
		sbragi.Trace("selected task", "type", t)
		if service != "" {
			fmt.Printf("sudo su - %s\n", service)
		}
		err = f(cfg, task, service)
		if sbragi.WithError(err).Trace("executed task") {
			return
		}
		if service == "" {
			cfg.System.Cluster.System[0].Tasks = feat.Tasks[1:]
		} else {
			cfg.System.Cluster.Services[0].Definition.Requirements.Features[0].Tasks = feat.Tasks[1:]
			fmt.Printf("exit\n")
		}
		state, err = Next(cfg)
		return
	}
}

func replaceAll(cfg pconf.Environment, s string) string {
	rs := []string{}
	if len(cfg.System.Cluster.System) == 0 && len(cfg.System.Cluster.Services) > 0 {
		rs = append(rs, "<service>", cfg.System.Cluster.Services[0].Name)
	}
	return strings.NewReplacer(rs...).Replace(s)
}

func replaceAllAll(cfg pconf.Environment, arr []string) []string {
	out := make([]string, len(arr))
	for i := range arr {
		out[i] = replaceAll(cfg, arr[i])
	}
	return out
}
