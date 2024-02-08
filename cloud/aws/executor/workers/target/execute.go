package target

import (
	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

var Fingerprint = adapter.New[[]loadbalancer.Target]("CreateTarget")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (ts []loadbalancer.Target, err error) {
		nodes := node.Fingerprint.Value(a[0])
		tgs := tg.Fingerprint.Value(a[1])
		ts = make([]loadbalancer.Target, len(nodes)*len(tgs))
		for i, tg := range tgs {
			off := i * len(nodes)
			for j, node := range nodes {
				var t loadbalancer.Target
				t, err = loadbalancer.CreateTarget(tg, node.Id, c)
				if log.WithError(err).Trace("while creating target", "node", node, "target_group", tg) {
					return
				}
				ts[off+j] = t
			}
		}
		return

	}, node.Fingerprint, tg.Fingerprint)
}
