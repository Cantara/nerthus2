package node

import (
	"sync"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

type Requireing interface {
	Node(server.Server) executor.Func
}

type data struct {
	c           *ec2.Client
	cluster     string
	names       []string
	port        int
	serviceType string
	system      string
	env         string
	size        string
	nerthus     string
	visuale     string
	subnets     []string
	img         *ami.Image
	key         *key.Key
	sg          *security.Group
	rs          []Requireing

	l *sync.Mutex
}

func Executor(nodes []string, port int, serviceType, cluster, system, env, size, nerthus, visuale string, rs []Requireing, c *ec2.Client) *data {
	/*
		names := make([]string, numNodes)
		for i := range names {
			names[i] = fmt.Sprintf("%s-%s-%d", env, cluster, i+1)
		}
	*/
	return &data{
		c:           c,
		names:       nodes,
		cluster:     cluster,
		port:        port,
		system:      system,
		serviceType: serviceType,
		env:         env,
		size:        size,
		nerthus:     nerthus,
		visuale:     visuale,
		rs:          rs,

		l: &sync.Mutex{},
	}
}

func (d *data) Execute(c chan<- executor.Func) {
	log.Debug("executing node")
	nodes := make([]server.Server, len(d.names))
	ids := make([]string, len(d.names))
	for i := range d.names {
		s, err := server.Create(i, d.names, d.port, d.serviceType, d.cluster, d.system, d.env, d.size, d.subnets[i], d.nerthus, d.visuale, *d.img, *d.key, *d.sg, d.c)
		if err != nil {
			log.WithError(err).Error("while creating nodes")
			c <- d.Execute
			return
		}
		nodes[i] = s
		ids[i] = s.Id
	}
	server.WaitUntilRunning(ids, d.c)
	for _, n := range nodes {
		for _, r := range d.rs {
			f := r.Node(n)
			if f == nil {
				continue
			}
			c <- f
		}
	}
}

func (d *data) Subnets(subnets []string) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.subnets = subnets
	return d.executable()
}

func (d *data) Image(i ami.Image) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.img = &i
	return d.executable()
}

func (d *data) Key(k key.Key) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.key = &k
	return d.executable()
}

func (d *data) SG(sg security.Group) executor.Func {
	defer d.l.Unlock()
	d.l.Lock()
	d.sg = &sg
	return d.executable()
}

func (d *data) executable() executor.Func {
	if len(d.subnets) == 0 {
		return nil
	}
	if d.img == nil {
		return nil
	}
	if d.key == nil {
		return nil
	}
	if d.sg == nil {
		return nil
	}

	return d.Execute
}
