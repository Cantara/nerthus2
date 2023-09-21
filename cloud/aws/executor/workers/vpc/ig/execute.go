package ig

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type data struct {
	c *ec2.Client
	v *vpc.VPC
}

func Executor(c *ec2.Client) *data {
	return &data{
		c: c,
	}
}

func (d *data) Execute() (any, error) {
	ig, err := vpc.NewIG(*d.v, d.c)
	if err != nil {
		log.WithError(err).Error("while creating ig")
		return nil, err
	}
	err = vpc.AddIGtoRT(d.v.Id, ig, d.c)
	if err != nil {
		log.WithError(err).Error("while adding ig to rt")
		return nil, err
	}
	return nil, nil
}

func (d *data) VPC(v vpc.VPC) {
	d.v = &v
}
