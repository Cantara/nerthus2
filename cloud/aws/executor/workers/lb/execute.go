package lb

import (
	"fmt"
	"sync"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/security"
)

type Requireing interface {
	LB(loadbalancer.Loadbalancer) executor.Func
}

type data struct {
	c       *elbv2.Client
	cluster string
	name    string
	env     string
	subnets []string
	sg      *security.Group
	rs      []Requireing

	l *sync.Mutex
}

func Executor(env, cluster string, rs []Requireing, c *elbv2.Client) *data {
	return &data{
		c:       c,
		name:    fmt.Sprintf("%s-%s-lb", env, cluster),
		cluster: cluster,
		env:     env,
		rs:      rs,

		l: &sync.Mutex{},
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	log.Trace("executing loadbalancer")

	lb, err := loadbalancer.CreateLoadbalancer(d.name, d.sg.Id, d.subnets, d.c)
	if err != nil {
		log.WithError(err).Error("while creating nodes")
		c <- d.Execute
		return
	}

	for _, r := range d.rs {
		f := r.LB(lb)
		if f == nil {
			continue
		}
		c <- f
	}
}

func (d *data) Subnets(subnets []string) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.subnets = subnets
	return d.executable()
}

func (d *data) SG(sg security.Group) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.sg = &sg
	return d.executable()
}

func (d *data) executable() executor.Func {
	if len(d.subnets) == 0 {
		return nil
	}
	if d.sg == nil {
		return nil
	}

	return d.Execute
}
