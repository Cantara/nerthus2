package sn

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type Requireing interface {
	Subnets([]string) executor.Func
}

type data struct {
	c  *ec2.Client
	v  vpc.VPC
	rs []Requireing
}

func Executor(rs []Requireing, c *ec2.Client) *data {
	return &data{
		c:  c,
		rs: rs,
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	err := vpc.CreateSubnets(d.v, d.c)
	if err != nil {
		log.WithError(err).Error("while getting subnets")
		c <- d.Execute
		return
	}
	s, err := vpc.GetSubnets(d.v.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while getting subnets")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.Subnets(vpc.SubnetsToIds(s))
		if f == nil {
			continue
		}
		c <- f
	}
}

func (d *data) VPC(v vpc.VPC) executor.Func {
	d.v = v
	return d.Execute
}
