package node

import (
	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/image"
	keye "github.com/cantara/nerthus2/cloud/aws/executor/workers/key"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

var Fingerprint = adapter.New[[]server.Server]("CreateOrGetNodes")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) ([]server.Server, error) {
		d := start.Fingerprint.Value(a[0])
		subnets := sn.Fingerprint.Value(a[1])
		img := image.Fingerprint.Value(a[2])
		k := keye.Fingerprint.Value(a[3])
		sg := sg.Fingerprint.Value(a[4])
		servs := make([]server.Server, len(d.Nodes))
		var ids []string
		for i := range d.Nodes {
			s, err := server.Create(i, d.Nodes[i], d.Cluster, d.System, d.Env, d.Size, subnets[i%len(subnets)], d.Nerthus, d.Visuale, img, k, sg, c)
			log.WithError(err).Trace("while creating nodes", "env", d.Env, "system", d.System, "cluster", d.Cluster, "image", img.HName, "subnets", subnets, "node", i)
			if err != nil {
				return nil, err
			}
			servs[i] = s
		}
		server.WaitUntilRunning(ids, c)
		return servs, nil
	}, start.Fingerprint, sn.Fingerprint, image.Fingerprint, keye.Fingerprint, sg.Fingerprint)
}

type data struct {
	c       *ec2.Client
	cluster string
	name    string
	num     int
	system  string
	env     string
	size    string
	nerthus string
	visuale string
	subnets []string
	img     *ami.Image
	key     *key.Key
	sg      *security.Group
}

func Executor(num int, node, cluster, system, env, size, nerthus, visuale string, c *ec2.Client) *data {
	return &data{
		c:       c,
		name:    node,
		num:     num,
		cluster: cluster,
		system:  system,
		env:     env,
		size:    size,
		nerthus: nerthus,
		visuale: visuale,
	}
}

func (d *data) Execute() (server.Server, error) {
	log.Debug("executing node")
	s, err := server.Create(d.num, d.name, d.cluster, d.system, d.env, d.size, d.subnets[d.num%len(d.subnets)], d.nerthus, d.visuale, *d.img, *d.key, *d.sg, d.c)
	if err != nil {
		log.WithError(err).Error("while creating nodes", "env", d.env, "system", d.system, "cluster", d.cluster, "image", d.img.HName, "subnets", d.subnets, "node", d.num)
		return server.Server{}, err
	}
	server.WaitUntilRunning([]string{s.Id}, d.c)
	return s, nil
}

func (d *data) Subnets(subnets []string) {
	d.subnets = subnets
}

func (d *data) Image(i ami.Image) {
	d.img = &i
}

func (d *data) Key(k key.Key) {
	d.key = &k
}

func (d *data) SG(sg security.Group) {
	d.sg = &sg
}
