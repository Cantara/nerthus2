package listener

import (
	"sync"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

type data struct {
	c    *elbv2.Client
	lb   *loadbalancer.Loadbalancer
	cert *acm.Cert

	l *sync.Mutex
}

func Executor(c *elbv2.Client) *data {
	return &data{
		c: c,

		l: &sync.Mutex{},
	}
}

func (d *data) Execute() (listner loadbalancer.Listener, err error) {
	log.Trace("executing listner")

	listner, err = loadbalancer.CreateListener(d.lb.ARN, d.cert.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating listner")
		return
	}
	return
}

func (d *data) Cert(c acm.Cert) {
	d.cert = &c
}

func (d *data) LB(lb loadbalancer.Loadbalancer) {
	d.lb = &lb
}
