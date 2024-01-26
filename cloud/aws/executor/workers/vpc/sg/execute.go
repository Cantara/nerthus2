package sg

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	vpce "github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/security"
)

var Fingerprint = adapter.New[security.Group]("CreateSecurityGroup")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (sg security.Group, err error) {
		env := start.Fingerprint.Value(a[0])
		v := vpce.Fingerprint.Value(a[1])
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		if env.System.MachineName != env.System.Cluster.MachineName {
			extra = fmt.Sprintf("%s-%s", extra, env.System.Cluster.MachineName)
		}
		name := fmt.Sprintf("%s%s-sg", env.MachineName, extra)
		sg, err = security.New(name, env.System.Cluster.Name, v.Id, c)
		log.WithError(err).Trace("creating new security group", "name", name, "vpc", v.Id)
		return

	}, start.Fingerprint, vpce.Fingerprint)
}
