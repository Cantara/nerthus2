package sg

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type data struct {
	c       *ec2.Client
	env     string
	system  string
	cluster string
	name    string
	v       vpc.VPC
}

func Executor(env, system, cluster string, c *ec2.Client) *data {
	return &data{
		c:       c,
		env:     env,
		system:  system,
		cluster: cluster,
		name:    fmt.Sprintf("%s-%s-%s-sg", env, system, cluster),
	}
}

func (d *data) Execute() (sg security.Group, err error) {
	sg, err = security.New(d.name, d.cluster, d.v.Id, d.c)
	if err != nil {
		log.WithError(err).Error("while creating new security group")
		return
	}
	return
}

func (d *data) VPC(v vpc.VPC) {
	d.v = v
}
