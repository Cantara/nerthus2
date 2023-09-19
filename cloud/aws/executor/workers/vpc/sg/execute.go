package sg

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type Requireing interface {
	SG(security.Group) executor.Func
}

type data struct {
	c       *ec2.Client
	env     string
	system  string
	cluster string
	name    string
	v       vpc.VPC
	rs      []Requireing
}

func Executor(env, system, cluster string, rs []Requireing, c *ec2.Client) *data {
	return &data{
		c:       c,
		env:     env,
		system:  system,
		cluster: cluster,
		name:    fmt.Sprintf("%s-%s-%s-sg", env, system, cluster),
		rs:      rs,
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	sg, err := security.New(d.name, d.cluster, d.v.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating new security group")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.SG(sg)
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
