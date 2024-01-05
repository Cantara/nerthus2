package vpc

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

var Fingerprint = adapter.New[vpc.VPC]("CreateOrGetVPC")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (v vpc.VPC, err error) {
		env := start.Fingerprint.Value(a[0])
		name := fmt.Sprintf("%s-%s-vpc", env.Name, env.System.Name)
		v, err = vpc.NewVPC(name, env.System.Cidr, c)
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
