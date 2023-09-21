package lb

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/security"
)

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
