package vpc

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/ig"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type Requireing interface {
	VPC(vpc.VPC) executor.Func
}

type data struct {
	c       *ec2.Client
	cluster string
	name    string
	network string
	rs      []Requireing
}

func Executor(env, cluster, network string, rs []Requireing, c *ec2.Client) data {
	return data{
		c:       c,
		cluster: cluster,
		name:    env + "-" + cluster + "-vpc",
		network: network,
		rs:      rs,
	}
}

func (d data) Execute(c chan<- executor.Func) {
	v, err := vpc.NewVPC(d.name, d.network, d.c)
	if err != nil {
		log.WithError(err).Error("while creating vpc")
		c <- d.Execute
		return
	}
	//TODO: This step could be skipped if v is not new
	v, err = vpc.GetVPC(d.name, d.c)
	if err != nil {
		log.WithError(err).Error("while getting vpc")
		c <- d.Execute
		return
	}
	c <- ig.Executor(v, d.c).Execute
	for _, r := range d.rs {
		f := r.VPC(v)
		if f == nil {
			continue
		}
		c <- f
	}
}
