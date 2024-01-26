package rule

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/listener"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/config/schema"
)

var Fingerprint = adapter.New[loadbalancer.Rule]("CreateRule")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (r loadbalancer.Rule, err error) {
		env := start.Fingerprint.Value(a[0])
		l := listener.Fingerprint.Value(a[1])
		tgs := tg.Fingerprint.Value(a[2])
		//For now do not split the cluster executions
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		if env.System.MachineName != env.System.Cluster.MachineName {
			extra = fmt.Sprintf("%s-%s", extra, env.System.Cluster.MachineName)
		}
		dnsName := fmt.Sprintf("%s%s.%s", env.MachineName, extra, env.System.Domain)
		for _, tg := range tgs {
			if env.System.RoutingMethod == schema.PathRouting {
				r, err = loadbalancer.CreateRulePath(l, tg, c)
			} else if env.System.Cluster.HasFrontend() {
				r, err = loadbalancer.CreateRuleDefault(l, tg, c)
			} else {
				r, err = loadbalancer.CreateRuleHost(l, tg, dnsName, c)
			}
			if log.WithError(err).Trace("while creating rule", "listener", l, "target_group", tg) {
				return
			}
		}
		return

	}, start.Fingerprint, listener.Fingerprint, tg.Fingerprint)
}
