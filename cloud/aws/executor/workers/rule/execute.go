package rule

import (
	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/listener"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

var Fingerprint = adapter.New[loadbalancer.Rule]("CreateRule")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (r loadbalancer.Rule, err error) {
		l := listener.Fingerprint.Value(a[0])
		t := tg.Fingerprint.Value(a[1])
		r, err = loadbalancer.CreateRule(l, t, c)
		log.WithError(err).Trace("while creating rule", "listener", l, "target_group", t)
		return

	}, listener.Fingerprint, tg.Fingerprint)
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

	_, err := loadbalancer.CreateRule(*d.listner, *d.tg, d.c)
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
