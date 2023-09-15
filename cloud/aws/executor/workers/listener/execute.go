package listener

import (
	"sync"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
)

type Requireing interface {
	Listener(loadbalancer.Listener) executor.Func
}

type data struct {
	c    *elbv2.Client
	lb   *loadbalancer.Loadbalancer
	cert *acm.Cert
	rs   []Requireing

	l *sync.Mutex
}

func Executor(rs []Requireing, c *elbv2.Client) *data {
	return &data{
		c:  c,
		rs: rs,

		l: &sync.Mutex{},
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	log.Trace("executing listner")

	listner, err := loadbalancer.CreateListener(d.lb.ARN, d.cert.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating listner")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.Listener(listner)
		if f == nil {
			continue
		}
		c <- f
	}
}

func (d *data) Cert(c acm.Cert) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.cert = &c
	return d.executable()
}

func (d *data) LB(lb loadbalancer.Loadbalancer) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.lb = &lb
	return d.executable()
}

func (d *data) executable() executor.Func {
	if d.cert == nil {
		return nil
	}
	if d.lb == nil {
		return nil
	}

	return d.Execute
}
