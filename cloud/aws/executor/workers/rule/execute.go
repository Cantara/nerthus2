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
	"github.com/cantara/nerthus2/system"
)

var Fingerprint = adapter.New[loadbalancer.Rule]("CreateRule")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (r loadbalancer.Rule, err error) {
		s := start.Fingerprint.Value(a[0])
		l := listener.Fingerprint.Value(a[1])
		t := tg.Fingerprint.Value(a[2])
		if s.Routing == system.RoutingPath {
			r, err = loadbalancer.CreateRulePath(l, t, c)
		} else if s.IsFrontend {
			r, err = loadbalancer.CreateRuleDefault(l, t, c)
		} else {
			r, err = loadbalancer.CreateRuleHost(l, t, fmt.Sprintf("%s-%s.%s", s.System, s.Cluster, s.Domain), c)
		}
		log.WithError(err).Trace("while creating rule", "listener", l, "target_group", t)
		return

	}, start.Fingerprint, listener.Fingerprint, tg.Fingerprint)
}

type data struct {
	c       *elbv2.Client
	listner *loadbalancer.Listener
	tg      *loadbalancer.TargetGroup
}

func Executor(c *elbv2.Client) *data {
	return &data{
		c: c,
	}
}

func (d *data) Execute() (any, error) {
	log.Trace("executing rule")

	_, err := loadbalancer.CreateRulePath(*d.listner, *d.tg, d.c)
	if err != nil {
		log.WithError(err).Error("while creating rule")
		return nil, err
	}
	return nil, nil
}

func (d *data) Listener(l loadbalancer.Listener) {
	d.listner = &l
}

func (d *data) TG(tg loadbalancer.TargetGroup) {
	d.tg = &tg
}
