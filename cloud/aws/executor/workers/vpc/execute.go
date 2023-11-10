package vpc

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

var Fingerprint = adapter.New[vpc.VPC]("CreateOrGetVPC")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (v vpc.VPC, err error) {
		d := start.Fingerprint.Value(a[0])
		name := d.Env + "-" + d.System + "-vpc"
		v, err = vpc.NewVPC(name, d.Network, c)
		if err != nil {
			log.WithError(err).Error("while creating vpc")
			return
		}
		//TODO: This step could be skipped if v is not new
		v, err = vpc.GetVPC(name, c)
		if err != nil {
			log.WithError(err).Error("while getting vpc")
			return
		}
		return

	}, start.Fingerprint)
}

type data struct {
	c       *ec2.Client
	system  string
	name    string
	network string
}

func Executor(env, system, network string, c *ec2.Client) data {
	return data{
		c:       c,
		system:  system,
		name:    env + "-" + system + "-vpc",
		network: network,
	}
}

func (d data) Execute() (v vpc.VPC, err error) {
	v, err = vpc.NewVPC(d.name, d.network, d.c)
	if err != nil {
		log.WithError(err).Error("while creating vpc")
		return
	}
	//TODO: This step could be skipped if v is not new
	v, err = vpc.GetVPC(d.name, d.c)
	if err != nil {
		log.WithError(err).Error("while getting vpc")
		return
	}
	return
}
