package tg

import (
	"fmt"
	"strings"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type data struct {
	c       *elbv2.Client
	env     string
	system  string
	cluster string
	name    string
	path    string
	port    int
	v       vpc.VPC
}

func Executor(env, system, cluster, path string, port int, c *elbv2.Client) *data {
	return &data{
		c:       c,
		env:     env,
		system:  system,
		cluster: cluster,
		name:    fmt.Sprintf("%s-%s-%s-tg", env, system, cluster), //This will be to long
		path:    fmt.Sprintf("/%s/health", strings.Trim(path, "/")),
		port:    port,
	}
}

func (d *data) Execute() (tg loadbalancer.TargetGroup, err error) {
	tg, err = loadbalancer.GetTargetGroup(d.name, d.path, d.port, d.c)
	if err == nil {
		return
	}
	tg, err = loadbalancer.CreateTargetGroup(d.v.Id, d.name, d.path, d.port, d.c)
	if err != nil {
		log.WithError(err).Error("while creating new target group")
		return
	}
	return
}

func (d *data) VPC(v vpc.VPC) {
	d.v = v
}
