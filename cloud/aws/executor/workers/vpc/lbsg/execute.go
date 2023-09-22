package lbsg

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type data struct {
	c      *ec2.Client
	env    string
	system string
	name   string
	v      vpc.VPC
}

func Executor(env, system string, c *ec2.Client) *data {
	return &data{
		c:      c,
		env:    env,
		system: system,
		name:   fmt.Sprintf("%s-%s-lb", env, system),
	}
}

func (d *data) Execute() (sg security.Group, err error) {
	sg, err = security.New(d.env, d.name, d.v.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating new security group")
		return
	}
	err = sg.AddLoadbalancerPublicAccess(d.c)
	if err != nil {
		log.WithError(err).Error("while setting public access to loadbalancer")
		return
	}
	return
}

func (d *data) VPC(v vpc.VPC) {
	d.v = v
}
