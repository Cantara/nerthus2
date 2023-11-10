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
		i := start.Fingerprint.Value(a[0])
		img, err = ami.GetImage(i.OSName, i.Arch, c)
		if err != nil {
			log.WithError(err).Error("while getting newest image")
			return
		}
		return
	}, start.Fingerprint)
}

type data struct {
	c    *ec2.Client
	name string
	arch ami.Arch
}

func Executor(name string, arch ami.Arch, c *ec2.Client) data {
	return data{
		c:    c,
		name: name,
		arch: arch,
	}
}

func (d data) Execute() (img ami.Image, err error) {
	img, err = ami.GetImage(d.name, d.arch, d.c)
	if err != nil {
		log.WithError(err).Error("while getting newest image")
		return
	}
	return
}
