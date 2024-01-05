package image

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
)

var Fingerprint = adapter.New[ami.Image]("GetImage")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (img ami.Image, err error) {
		env := start.Fingerprint.Value(a[0])
		img, err = ami.GetImage(env.System.Cluster.Node.Os.Name, env.System.Cluster.Node.Arch, c)
		if err != nil {
			log.WithError(err).Error("while getting newest image")
			return
		}
		return
	}, start.Fingerprint)
}
