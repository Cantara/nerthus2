package tg

import (
	"fmt"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type Requireing interface {
	TG(loadbalancer.TargetGroup) executor.Func
}

type data struct {
	c       *elbv2.Client
	env     string
	system  string
	cluster string
	name    string
	path    string
	port    int
	v       vpc.VPC
	rs      []Requireing
}

func Executor(env, system, cluster, path string, port int, rs []Requireing, c *elbv2.Client) *data {
	return &data{
		c:       c,
		env:     env,
		system:  system,
		cluster: cluster,
		name:    fmt.Sprintf("%s-%s-%s-tg", env, system, cluster), //This will be to long
		path:    path,
		port:    port,
		rs:      rs,
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	tg, err := loadbalancer.CreateTargetGroup(d.v.Id, d.name, d.path, d.port, d.c)
	if err != nil {
		log.WithError(err).Error("while creating new target group")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.TG(tg)
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
