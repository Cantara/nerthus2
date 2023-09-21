package workers

import (
	"fmt"
	"strings"

	awsacm "github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/cert"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/image"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/key"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/lb"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/listener"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/rule"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/saga"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/target"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/ig"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/lbsg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
	awskey "github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/server"
	awsvpc "github.com/cantara/nerthus2/cloud/aws/vpc"
	"github.com/cantara/nerthus2/system"
)

type Executor interface {
	Add(executor.Func)
}

func Deploy(sys system.System, env, nerthus, visuale string, c chan<- saga.Executable, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *awsacm.Client) { //TODO: Change fingerprint to take inn config object
	s := &saga.Saga{}
	ve := &saga.Event[awsvpc.VPC]{Func: vpc.Executor(env, sys.Name, sys.CIDR, e2).Execute}
	s.Event(ve)

	ige := ig.Executor(e2)
	igse := &saga.Event[any]{Func: ige.Execute}
	ve.Mandates(saga.Mandatabale(igse, ige.VPC))
	igse.Mandates(s.End)

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
			//Move the path creation to config parsing
			tge := tg.Executor(env, sys.Name, cluster.Name, fmt.Sprintf("/%s", strings.ToLower(service.ServiceInfo.Artifact.Id)), *service.WebserverPort, elb)
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
				tse.Mandates(s.End)
			}
		}
		prse.Mandates(s.End)
	}

	s.Execute(c)
}

/*
func DeployInfra(nodes []string, arch ami.Arch, imageName, network, cluster, system, env, size, nerthus, visuale, domain string, e Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client) { //TODO: Change fingerprint to take inn config object
	te := target.Executor(elb)
	re := rule.Executor(elb)

	le := listener.Executor([]listener.Requireing{
		re,
	}, elb)

	e.Add(cert.Executor(domain, []cert.Requireing{
		le,
	}, cc, rc).Execute)

	lbe := lb.Executor(env, system, []lb.Requireing{
		le,
	}, elb)

	ne := node.Executor(nodes, cluster, system, env, size, nerthus, visuale, []node.Requireing{
		te,
	}, e2)
	e.Add(key.Executor(env, system, []key.Requireing{
		ne,
	}, e2).Execute)
	e.Add(image.Executor(imageName, arch, []image.Requireing{
		ne,
	}, e2).Execute)

	lbsge := lbsg.Executor(env, system, []lbsg.Requireing{
		lbe,
	}, e2)
	sne := sn.Executor([]sn.Requireing{
		ne,
		lbe,
	}, e2)
	sge := sg.Executor(env, system, cluster, []sg.Requireing{
		ne,
	}, e2)
	/*
		tge := tg.Executor(env, cluster, path, port, []tg.Requireing{
			te,
			re,
		}, elb)
*/ /*

	e.Add(vpc.Executor(env, system, network, []vpc.Requireing{
		lbsge,
		sne,
		sge,
		//tge,
	}, e2).Execute)
}
*/
