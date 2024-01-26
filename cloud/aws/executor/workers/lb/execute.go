package lb

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/lbsg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

var Fingerprint = adapter.New[loadbalancer.Loadbalancer]("CreateLoadbalancer")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (lb loadbalancer.Loadbalancer, err error) {
		env := start.Fingerprint.Value(a[0])
		subnets := sn.Fingerprint.Value(a[1])
		sg := lbsg.Fingerprint.Value(a[2])
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		name := fmt.Sprintf("%s%s-lb", env.MachineName, extra)
		lb, err = loadbalancer.CreateLoadbalancer(name, sg.Id, subnets, c)
		log.WithError(err).Trace("creating loadbalancer", "name", name, "subnets", subnets)
		return lb, err

	}, start.Fingerprint, sn.Fingerprint, lbsg.Fingerprint)
}
