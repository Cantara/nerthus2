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
		tgs = make([]loadbalancer.TargetGroup, len(env.System.Cluster.Services))
		for i, service := range env.System.Cluster.Services {
			name := fmt.Sprintf("%s-%s-%s-%s-tg", env.Name, env.System.Name, env.System.Cluster.Name, service.MachineName) //This will be to long
			path := fmt.Sprintf("/%s/health", strings.Trim(service.Definition.APIPath, "/"))
			tgs[i], err = loadbalancer.GetTargetGroup(name, path, service.Port, c)
			if err == nil {
				continue
			}
			tgs[i], err = loadbalancer.CreateTargetGroup(v.Id, name, path, service.Port, c)
			if log.WithError(err).Trace("creating new target group", "name", name, "vpc", v.Id) {
				return
			}
		}
		return

	}, start.Fingerprint, vpce.Fingerprint)
}
