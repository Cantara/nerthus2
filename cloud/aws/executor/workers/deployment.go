package workers

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/cantara/bragi/sbragi"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/consensus"
	"github.com/cantara/gober/stream"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/cert"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/reader"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/story"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/image"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/key"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/lb"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/listener"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/rule"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/target"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/ig"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/lbsg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
	"github.com/cantara/nerthus2/system"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

type Provisioner interface {
	Provision(env system.Environment)
	ProvisionSystem(sys system.System, env, nerthus, visuale string)
	ProvisionCluster(clust system.Cluster, sys system.System, env, nerthus, visuale string)
	Work()
}

type provisioner struct {
	r reader.Reader
}

func New(strm stream.Stream, cb consensus.ConsBuilderFunc, cryptoKey string, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, ac *acm.Client, ctx context.Context) (Provisioner, error) {
	vpcA := vpc.Adapter(e2)
	keyA := key.Adapter(e2)
	imgA := image.Adapter(e2)
	certA := cert.Adapter(ac, rc)
	lbsgA := lbsg.Adapter(e2)
	snA := sn.Adapter(e2)
	igA := ig.Adapter(e2)
	tgA := tg.Adapter(elb)
	sgA := sg.Adapter(e2)
	nodeA := node.Adapter(e2)
	lbA := lb.Adapter(elb)
	lsA := listener.Adapter(elb)
	rA := rule.Adapter(elb)
	tA := target.Adapter(elb)
	/*
		Take full adapter into story and use adapter to verify all LinkTo is met. Aswell as using the adapternames in creating and returning data from stream.
		This should use simple names so that the adapter easily can be spoofed in sub stories.
		Use fingerprints structs for adapters to define names and return values. This can then be imported and used for requirements and srict json matching
	*/
	s, err := story.Start("deploy").
		LinkTo("vpc", "key", "image", "cert", "node", "lbsg", "tg", "lb", "sg").
		Id("vpc").Adapter(vpcA.Name()).LinkTo("lbsg", "sn", "ig", "tg", "sg").
		Id("key").Adapter(keyA.Name()).LinkTo("node").
		Id("image").Adapter(imgA.Name()).LinkTo("node").
		Id("cert").Adapter(certA.Name()).LinkTo("listener").
		Id("lbsg").Adapter(lbsgA.Name()).LinkTo("lb").
		Id("sn").Adapter(snA.Name()).LinkTo("lb", "node").
		Id("ig").Adapter(igA.Name()).LinkToEnd().
		Id("tg").Adapter(tgA.Name()).LinkTo("rule", "target").
		Id("sg").Adapter(sgA.Name()).LinkTo("node").
		Id("node").Adapter(nodeA.Name()).LinkTo("target").
		Id("lb").Adapter(lbA.Name()).LinkTo("listener").
		Id("listener").Adapter(lsA.Name()).LinkTo("rule").
		Id("rule").Adapter(rA.Name()).LinkToEnd().
		Id("target").Adapter(tA.Name()).LinkToEnd().End()
	if err != nil {
		return nil, err
	}
	r, err := reader.New[start.Start](strm, cb, stream.StaticProvider(sbragi.RedactedString(cryptoKey)), time.Minute*5, s, ctx, start.Adapter, vpcA, keyA, imgA, certA, lbsgA, snA, igA, tgA, sgA, nodeA, lbA, lsA, rA, tA)
	if err != nil {
		return nil, err
	}

	return provisioner{
		r: r,
	}, nil
}

func (d provisioner) Provision(env system.Environment) {
	for _, sys := range env.SystemConfigs {
		d.ProvisionSystem(sys, env.Name, env.Nerthus, env.Visuale)
	}

}

func (d provisioner) ProvisionSystem(sys system.System, env, nerthus, visuale string) {
	for _, cluster := range sys.Clusters {
		d.ProvisionCluster(*cluster, sys, env, nerthus, visuale)
	}

}

func (d provisioner) ProvisionCluster(cluster system.Cluster, sys system.System, env, nerthus, visuale string) {
	a, err := ami.StringToArch(sys.OSArch)
	if err != nil {
		log.WithError(err).Fatal("os arch should be verified before here")
	}
	p := start.Start{
		Env:     env,
		System:  sys.Name,
		Cluster: cluster.Name,
		OSName:  sys.OSName,
		Arch:    a,
		Network: sys.CIDR,
		Nodes:   cluster.NodeNames,
		Size:    cluster.InstanceType,
		Nerthus: nerthus,
		Visuale: visuale,
		Path:    cluster.GetWebserverPath(), //FixMe: This needs to be fixed
		Port:    cluster.GetWebserverPort(),
		Base:    cluster.DNSRoot,
	}
	b, err := json.Marshal(p)
	if err != nil {
		log.WithError(err).Fatal("json should not fail")
	}
	d.r.New(b)
	log.Info("new", "b", string(b))

}

func (d provisioner) Work() {
	d.r.Read()
}

/*
type Executor interface {
	Add(executor.Func)
}

func Provision(sys system.System, env, nerthus, visuale string, c chan<- saga.Executable, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *awsacm.Client) { //TODO: Change fingerprint to take inn config object
	s := &saga.Saga{}
	ve := &saga.Event[awsvpc.VPC]{Func: vpc.Executor(env, sys.Name, sys.CIDR, e2).Execute}
	s.Event(ve)

	ige := ig.Executor(e2)
	igse := &saga.Event[any]{Func: ige.Execute}
	ve.Mandates(saga.Mandatabale(igse, ige.VPC))
	igse.Mandates(s.End())

	sne := sn.Executor(e2)
	snse := saga.Event[[]string]{Func: sne.Execute}
	ve.Mandates(saga.Mandatabale(&snse, sne.VPC))

	lbsge := lbsg.Executor(env, sys.Name, e2)
	lbsgse := &saga.Event[security.Group]{Func: lbsge.Execute}
	ve.Mandates(saga.Mandatabale(lbsgse, lbsge.VPC))

	lbe := lb.Executor(env, sys.Name, elb)
	lbse := &saga.Event[loadbalancer.Loadbalancer]{Func: lbe.Execute}
	snse.Mandates(saga.Mandatabale(lbse, lbe.Subnets))
	lbsgse.Mandates(saga.Mandatabale(lbse, lbe.SG))

	ce := &saga.Event[acm.Cert]{Func: cert.Executor(sys.Domain, cc, rc).Execute}
	s.Event(ce)

	le := listener.Executor(elb)
	lse := &saga.Event[loadbalancer.Listener]{Func: le.Execute}
	ce.Mandates(saga.Mandatabale(lse, le.Cert))
	lbse.Mandates(saga.Mandatabale(lse, le.LB))

	ke := &saga.Event[awskey.Key]{Func: key.Executor(env, sys.Name, e2).Execute}
	s.Event(ke)

	for _, cluster := range sys.Clusters {
		imge := &saga.Event[ami.Image]{Func: image.Executor(cluster.OSName, cluster.Arch, e2).Execute}
		s.Event(imge)

		sge := sg.Executor(env, sys.Name, cluster.Name, e2)
		sgse := &saga.Event[security.Group]{Func: sge.Execute}
		ve.Mandates(saga.Mandatabale(sgse, sge.VPC))

		nes := make([]*saga.Event[server.Server], len(cluster.NodeNames))
		for i, name := range cluster.NodeNames {
			n := node.Executor(i, name, cluster.Name, sys.Name, env, cluster.InstanceType, nerthus, visuale, e2)
			ne := &saga.Event[server.Server]{Func: n.Execute}
			sgse.Mandates(saga.Mandatabale(ne, n.SG))
			ke.Mandates(saga.Mandatabale(ne, n.Key))
			imge.Mandates(saga.Mandatabale(ne, n.Image))
			snse.Mandates(saga.Mandatabale(ne, n.Subnets))
			nes[i] = ne
		}

		var prse *saga.Event[any]
		for _, service := range cluster.Services {
			tge := tg.Executor(env, sys.Name, cluster.Name, service.ServiceInfo.APIPath, *service.WebserverPort, elb)
			tgse := &saga.Event[loadbalancer.TargetGroup]{Func: tge.Execute}
			ve.Mandates(saga.Mandatabale(tgse, tge.VPC))

			re := rule.Executor(elb)
			rse := &saga.Event[any]{Func: re.Execute}
			lse.Mandates(saga.Mandatabale(rse, re.Listener))
			tgse.Mandates(saga.Mandatabale(rse, re.TG))
			if prse != nil {
				prse.Mandates(saga.Mandatabale(rse, func(any) {}))
			}
			prse = rse

			for _, ne := range nes {
				te := target.Executor(elb)
				tse := &saga.Event[any]{Func: te.Execute}
				ne.Mandates(saga.Mandatabale(tse, te.Node))
				tgse.Mandates(saga.Mandatabale(tse, te.TG))
				tse.Mandates(s.End())
			}
		}
		prse.Mandates(s.End())
	}

	s.Execute(c)
}
*/
