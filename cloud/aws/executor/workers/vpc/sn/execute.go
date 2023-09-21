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
	c *ec2.Client
	v vpc.VPC
}

func Executor(c *ec2.Client) *data {
	return &data{
		c: c,
	}
}

func (d *data) Execute() ([]string, error) {
	err := vpc.CreateSubnets(d.v, d.c)
	if err != nil {
		log.WithError(err).Error("while getting subnets")
		return []string{}, err
	}
	s, err := vpc.GetSubnets(d.v.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while getting subnets")
		return []string{}, err
	}
	return vpc.SubnetsToIds(s), nil
}

func (d *data) VPC(v vpc.VPC) {
	d.v = v
}
