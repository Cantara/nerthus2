package image

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor"
)

type Requireing interface {
	Image(ami.Image) executor.Func
}

type data struct {
	c    *ec2.Client
	name string
	arch ami.Arch
	rs   []Requireing
}

func Executor(name string, arch ami.Arch, rs []Requireing, c *ec2.Client) data {
	return data{
		c:    c,
		name: name,
		arch: arch,
		rs:   rs,
	}
}

func (d data) Execute(c chan<- executor.Func) {
	img, err := ami.GetImage(d.name, d.arch, d.c)
	if err != nil {
		log.WithError(err).Error("while getting newest image")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.Image(img)
		if f == nil {
			continue
		}
		c <- f
	}
}
