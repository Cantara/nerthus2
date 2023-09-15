package ig

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type data struct {
	c *ec2.Client
	v vpc.VPC
}

func Executor(v vpc.VPC, c *ec2.Client) data {
	return data{
		c: c,
		v: v,
	}
}

func (d data) Execute(c chan<- executor.Func) {
	ig, err := vpc.NewIG(d.v, d.c)
	if err != nil {
		log.WithError(err).Error("while creating ig")
		c <- d.Execute
		return
	}
	err = vpc.AddIGtoRT(d.v.Id, ig, d.c)
	if err != nil {
		log.WithError(err).Error("while adding ig to rt")
		c <- d.Execute
		return
	}
}
