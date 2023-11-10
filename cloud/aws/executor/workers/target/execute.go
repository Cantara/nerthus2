package target

import (
	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

var Fingerprint = adapter.New[[]loadbalancer.Target]("CreateTarget")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (ts []loadbalancer.Target, err error) {
		nodes := node.Fingerprint.Value(a[0])
		tg := tg.Fingerprint.Value(a[1])
		ts = make([]loadbalancer.Target, len(nodes))
		for i, node := range nodes {
			var t loadbalancer.Target
			t, err = loadbalancer.CreateTarget(tg, node.Id, c)
			log.WithError(err).Trace("while creating target", "node", node, "target_group", tg)
			if err != nil {
				return
			}
			ts[i] = t
		}
		return

	}, node.Fingerprint, tg.Fingerprint)
}

type data struct {
	c  *elbv2.Client
	n  *server.Server
	tg *loadbalancer.TargetGroup
}

func Executor(c *elbv2.Client) *data {
	return &data{
		c: c,
	}
}

func (d *data) Execute() (any, error) {
	log.Debug("executing target")

	_, err := loadbalancer.CreateTarget(*d.tg, d.n.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating target")
		return nil, err
	}
	return nil, nil
}

func (d *data) TG(tg loadbalancer.TargetGroup) {
	d.tg = &tg
}

func (d *data) Node(n server.Server) {
	d.n = &n
}
