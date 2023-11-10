package lb

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/lbsg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/security"
)

var Fingerprint = adapter.New[loadbalancer.Loadbalancer]("CreateLoadbalancer")

func Adapter(c *elbv2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (lb loadbalancer.Loadbalancer, err error) {
		i := start.Fingerprint.Value(a[0])
		subnets := sn.Fingerprint.Value(a[1])
		sg := lbsg.Fingerprint.Value(a[2])
		name := fmt.Sprintf("%s-%s-lb", i.Env, i.System)
		lb, err = loadbalancer.CreateLoadbalancer(name, sg.Id, subnets, c)
		log.WithError(err).Trace("creating loadbalancer", "name", name, "subnets", subnets)
		return lb, err

	}, start.Fingerprint, sn.Fingerprint, lbsg.Fingerprint)
}

type data struct {
	c       *elbv2.Client
	system  string
	name    string
	env     string
	subnets []string
	sg      *security.Group
}

func Executor(env, system string, c *elbv2.Client) *data {
	return &data{
		c:      c,
		name:   fmt.Sprintf("%s-%s-lb", env, system),
		system: system,
		env:    env,
	}
}

func (d *data) Execute() (lb loadbalancer.Loadbalancer, err error) {
	log.Trace("executing loadbalancer")

	lb, err = loadbalancer.CreateLoadbalancer(d.name, d.sg.Id, d.subnets, d.c)
	if err != nil {
		log.WithError(err).Error("while creating loadbalancer", "name", d.name, "subnets", d.subnets)
		return
	}
	return
}

func (d *data) Subnets(subnets []string) {
	d.subnets = subnets
}

func (d *data) SG(sg security.Group) {
	d.sg = &sg
}
