package key

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/key"
)

var Fingerprint = adapter.New[key.Key]("NewKey")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (k key.Key, err error) {
		env := start.Fingerprint.Value(a[0])
		k, err = key.New(fmt.Sprintf("%s-%s-key", env.Name, env.System.Name), c)
		log.WithError(err).Trace("creating key")
		return
	}, start.Fingerprint)
}

type data struct {
	c      *ec2.Client
	env    string
	system string
	name   string
}

func Executor(env, system string, c *ec2.Client) data {
	return data{
		c:      c,
		env:    env,
		system: system,
		name:   fmt.Sprintf("%s-%s-key", env, system),
	}
}

func (d data) Execute() (k key.Key, err error) {
	k, err = key.New(d.name, d.c)
	if err != nil {
		log.WithError(err).Error("while creating key")
		return
	}
	return
}
