package key

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/key"
)

type Requireing interface {
	Key(key.Key) executor.Func
}

type data struct {
	c      *ec2.Client
	env    string
	system string
	name   string
	rs     []Requireing
}

func Executor(env, system string, rs []Requireing, c *ec2.Client) data {
	return data{
		c:      c,
		env:    env,
		system: system,
		name:   fmt.Sprintf("%s-%s-key", env, system),
		rs:     rs,
	}
}

func (d data) Execute(c chan<- executor.Func) {
	k, err := key.New(d.name, d.c)
	if err != nil {
		log.WithError(err).Error("while creating key")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.Key(k)
		if f == nil {
			continue
		}
		c <- f
	}
}
