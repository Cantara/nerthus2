package sn

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	vpce "github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

var Fingerprint = adapter.New[[]string]("CreateSubnets")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (sn []string, err error) {
		v := vpce.Fingerprint.Value(a[0])
		err = vpc.CreateSubnets(v, c)
		if err != nil {
			log.WithError(err).Error("while getting subnets")
			return []string{}, err
		}
		s, err := vpc.GetSubnets(v.Id, c)
		if err != nil {
			log.WithError(err).Error("while getting subnets")
			return []string{}, err
		}
		return vpc.SubnetsToIds(s), nil

	}, vpce.Fingerprint)
}

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
