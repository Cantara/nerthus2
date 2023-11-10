package listener

import (
	"sync"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/cert"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/lb"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

var Fingerprint = adapter.New[loadbalancer.Listener]("CreateListener")

func Adapter(elb *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (listner loadbalancer.Listener, err error) {
		c := cert.Fingerprint.Value(a[0])
		l := lb.Fingerprint.Value(a[1])
		listner, err = loadbalancer.CreateListener(l.ARN, c.Id, elb)
		log.WithError(err).Trace("while creating listner", "arn", l.ARN, "cert", c.Id)
		return

	}, cert.Fingerprint, lb.Fingerprint)
}

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
