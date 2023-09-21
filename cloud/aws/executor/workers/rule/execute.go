package rule

import (
	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

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
