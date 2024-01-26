package lbsg

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	vpce "github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/security"
)

var Fingerprint = adapter.New[security.Group]("CreateLoadbalancerSecurityGroup")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (sg security.Group, err error) {
		env := start.Fingerprint.Value(a[0])
		v := vpce.Fingerprint.Value(a[1])
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		name := fmt.Sprintf("%s%s-lb", env.MachineName, extra)
		sg, err = security.New(env.Name, name, v.Id, c)
		if err != nil {
			log.WithError(err).Error("while creating new security group")
			return
		}
		err = sg.AddLoadbalancerPublicAccess(c)
		if err != nil {
			log.WithError(err).Error("while setting public access to loadbalancer")
			return
		}
		return

	}, start.Fingerprint, vpce.Fingerprint)
}
