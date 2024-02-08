package tg

import (
	"fmt"
	"strings"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	vpce "github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

var Fingerprint = adapter.New[[]loadbalancer.TargetGroup]("CreateTargetGroup")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (tgs []loadbalancer.TargetGroup, err error) {
		env := start.Fingerprint.Value(a[0])
		v := vpce.Fingerprint.Value(a[1])
		log.Info("creating target groups", "a", a, "i", env, "v", v)
		//tgs = make([]loadbalancer.TargetGroup, len(env.System.Cluster.Services))
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		if env.System.MachineName != env.System.Cluster.MachineName {
			extra = fmt.Sprintf("%s-%s", extra, env.System.Cluster.MachineName)
		}
		for _, service := range env.System.Cluster.Services {
			if service.Port == 0 {
				log.Info("skipping creation of TG as port is 0", "service", service.Name)
				continue
			}
			extra := extra
			if env.System.Cluster.MachineName != service.MachineName {
				extra = fmt.Sprintf("%s-%s", extra, service.MachineName)
			}
			name := fmt.Sprintf("%s%s-tg", env.MachineName, extra)
			path := strings.ReplaceAll(fmt.Sprintf("/%s/health", strings.Trim(service.Definition.APIPath, "/")), "//", "/")
			var tg loadbalancer.TargetGroup
			log.Info("creating target group", "service", service.Name, "name", name, "path", path, "port", service.Port)
			tg, err = loadbalancer.GetTargetGroup(name, c)
			if err == nil {
				if tg.Path != path || tg.Port != service.Port {
					tg.Path = path
					tg.Port = service.Port
					tg.Update(c)
				}
				tgs = append(tgs, tg)
				continue
			}
			tg, err = loadbalancer.CreateTargetGroup(v.Id, name, path, service.Port, c)
			if log.WithError(err).Trace("creating new target group", "name", name, "vpc", v.Id) {
				return
			}
			tgs = append(tgs, tg)
		}
		return

	}, start.Fingerprint, vpce.Fingerprint)
}
