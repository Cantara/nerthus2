package target

import (
	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

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
