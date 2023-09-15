package target

import (
	"sync"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

type data struct {
	c  *elbv2.Client
	n  *server.Server
	tg *loadbalancer.TargetGroup

	l *sync.Mutex
}

func Executor(c *elbv2.Client) *data {
	return &data{
		c: c,

		l: &sync.Mutex{},
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	log.Debug("executing target")

	_, err := loadbalancer.CreateTarget(*d.tg, d.n.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating nodes")
		c <- d.Execute
		return
	}
}

func (d *data) TG(tg loadbalancer.TargetGroup) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.tg = &tg
	return d.executable()
}

func (d *data) Node(n server.Server) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.n = &n
	return d.executable()
}

func (d *data) executable() executor.Func {
	if d.n == nil {
		return nil
	}
	if d.tg == nil {
		return nil
	}

	return d.Execute
}
