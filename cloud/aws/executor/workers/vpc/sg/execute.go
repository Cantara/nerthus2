package sg

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	vpce "github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type inn struct {
	Env     string `json:"env"`
	System  string `json:"system"`
	Cluster string `json:"cluster"`
}

var Fingerprint = adapter.New[security.Group]("CreateSecurityGroup")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (sg security.Group, err error) {
		i := start.Fingerprint.Value(a[0])
		v := vpce.Fingerprint.Value(a[1])
		name := fmt.Sprintf("%s-%s-%s-sg", i.Env, i.System, i.Cluster)
		sg, err = security.New(name, i.Cluster, v.Id, c)
		log.WithError(err).Trace("creating new security group", "name", name, "vpc", v.Id)
		return

	}, start.Fingerprint, vpce.Fingerprint)
}

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
